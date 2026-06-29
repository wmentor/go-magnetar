package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
)

// contentUUID derives a deterministic UUID v5 from the text content.
// Using uuid.NameSpaceDNS as the namespace gives a stable, collision-resistant
// identifier: identical content always maps to the same UUID, so upserting the
// same chunk twice is idempotent (Qdrant overwrites the existing point).
func contentUUID(content string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(content)).String()
}

const (
	payloadTextField = "text"
	defaultGRPCPort  = 6334
	defaultTimeout   = time.Second * 30
)

// RAGTools provides RAG operations as LLM tools.
type RAGTools struct {
	cfg          *config.Config
	embedClient  *openai.Client
	llmClient    *openai.Client // used for query expansion (multi-query)
	qdrantClient *qdrant.Client
}

// New creates a new RAGTools instance, connects to Qdrant and ensures the collection exists.
func New(cfg *config.Config) (*RAGTools, error) {
	// Build OpenAI embedding client.
	embedCfg := openai.DefaultConfig(cfg.String("rag.llm.api_key"))
	embedCfg.BaseURL = cfg.String("rag.llm.base_url")
	embedClient := openai.NewClientWithConfig(embedCfg)

	// Build LLM client for query expansion (reuses main LLM config).
	llmCfg := openai.DefaultConfig(cfg.String("llm.api_key"))
	llmCfg.BaseURL = cfg.String("llm.base_url")
	llmClient := openai.NewClientWithConfig(llmCfg)

	// Parse connstr to extract host and port for gRPC.
	host, port, err := parseConnStr(cfg.String("rag.qdrant.connstr"))
	if err != nil {
		return nil, fmt.Errorf("rag: invalid qdrant connstr %q: %w", cfg.String("rag.qdrant.connstr"), err)
	}

	// Connect to Qdrant via gRPC.
	qdrantClient, err := qdrant.NewClient(&qdrant.Config{
		Host:                   host,
		Port:                   port,
		SkipCompatibilityCheck: true,
	})
	if err != nil {
		return nil, fmt.Errorf("rag: failed to connect to qdrant: %w", err)
	}

	rt := &RAGTools{
		cfg:          cfg,
		embedClient:  embedClient,
		llmClient:    llmClient,
		qdrantClient: qdrantClient,
	}

	// Ensure collection exists.
	if err := rt.ensureCollection(); err != nil {
		return nil, err
	}

	return rt, nil
}

// parseConnStr parses a URL like http://localhost:6333 and returns host and gRPC port (6334).
func parseConnStr(connStr string) (string, int, error) {
	u, err := url.Parse(connStr)
	if err != nil {
		return "", 0, err
	}

	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}

	// If port is explicitly specified in connstr, use it.
	// Otherwise use default gRPC port.
	port := defaultGRPCPort
	if p := u.Port(); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil {
			return "", 0, fmt.Errorf("invalid port %q: %w", p, err)
		}
		// REST port is typically 6333, gRPC is 6334.
		// If user provided REST port, switch to gRPC port.
		if parsed == 6333 {
			port = defaultGRPCPort
		} else {
			port = parsed
		}
	}

	return host, port, nil
}

// ensureCollection creates the Qdrant collection if it does not exist.
func (r *RAGTools) ensureCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	exists, err := r.qdrantClient.CollectionExists(ctx, r.cfg.String("rag.qdrant.collection"))
	if err != nil {
		return fmt.Errorf("rag: failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	vectorSize := uint64(r.cfg.Int("rag.llm.vector_size"))
	err = r.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: r.cfg.String("rag.qdrant.collection"),
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("rag: failed to create collection %q: %w", r.cfg.String("rag.qdrant.collection"), err)
	}

	printer.Info("rag: collection created", "collection", r.cfg.String("rag.qdrant.collection"))
	return nil
}

// embed returns the embedding vector for the given text.
func (r *RAGTools) embed(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := r.embedClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(r.cfg.String("rag.llm.model")),
	})
	if err != nil {
		return nil, fmt.Errorf("rag: embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("rag: empty embedding response")
	}

	return resp.Data[0].Embedding, nil
}

// expandQuery asks the LLM to produce n alternative phrasings of the query.
// Returns the reformulations (not including the original query).
// On any error it logs a warning and returns nil so the caller falls back to
// single-query mode gracefully.
func (r *RAGTools) expandQuery(ctx context.Context, query string, n int) []string {
	if n <= 0 {
		return nil
	}

	prompt := fmt.Sprintf(
		"Generate %d alternative phrasings of the following search query. "+
			"Output only the phrasings, one per line, with no numbering or extra text.\n\nQuery: %s",
		n, query,
	)

	reqCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := r.llmClient.CreateChatCompletion(reqCtx, openai.ChatCompletionRequest{
		Model: r.cfg.String("llm.model"),
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		printer.ToolCall(printer.IconAlert, "rag_search: query expansion failed, falling back to single query", "err", err)
		return nil
	}

	if len(resp.Choices) == 0 {
		return nil
	}

	var result []string
	for _, line := range strings.Split(resp.Choices[0].Message.Content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	printer.ToolCall(printer.IconSearch, "rag_search: query expansion", "original", preview(query, 60), "variants", len(result))
	return result
}

// searchResult holds one Qdrant result point with its vector for deduplication.
type searchResult struct {
	id     string
	score  float32
	text   string
	vector []float32
}

// searchOne executes a single embedding + Qdrant query and returns raw results.
func (r *RAGTools) searchOne(ctx context.Context, query string) ([]searchResult, error) {
	vector, err := r.embed(query)
	if err != nil {
		return nil, err
	}

	qCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	limit := uint64(r.cfg.Int("rag.search.limit"))
	withPayload := true
	scoreThreshold := float32(r.cfg.Float64("rag.search.threshold"))
	points, err := r.qdrantClient.Query(qCtx, &qdrant.QueryPoints{
		CollectionName: r.cfg.String("rag.qdrant.collection"),
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		ScoreThreshold: &scoreThreshold,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, err
	}

	var out []searchResult
	for _, p := range points {
		if payload := p.GetPayload(); payload != nil {
			if val, ok := payload[payloadTextField]; ok {
				if sv := val.GetStringValue(); sv != "" {
					out = append(out, searchResult{
						id:     p.GetId().GetUuid(),
						score:  p.GetScore(),
						text:   sv,
						vector: vector, // reuse query vector for inter-result dedup
					})
				}
			}
		}
	}
	return out, nil
}

// cosineSimilarity computes the cosine similarity between two unit-normalised
// vectors. Qdrant returns cosine-distance results, so vectors from different
// queries may not be unit-normalised — we normalise here.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// dedup removes near-duplicate chunks from results.
// Two chunks are considered near-duplicates when their text embeddings have a
// cosine similarity above cfg.Float64("rag.search.dedup_threshold"). When a pair is found,
// the chunk with the lower score is dropped.
// Embeddings for each unique chunk are computed lazily and cached within the call.
func (r *RAGTools) dedup(results []searchResult) []searchResult {
	threshold := r.cfg.Float64("rag.search.dedup_threshold")
	if threshold <= 0 || len(results) <= 1 {
		return results
	}

	// Compute embeddings for each chunk text (in parallel).
	embeddings := make([][]float32, len(results))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, res := range results {
		wg.Add(1)
		go func(idx int, text string) {
			defer wg.Done()
			vec, err := r.embed(text)
			if err != nil {
				printer.ToolCall(printer.IconAlert, "rag: dedup embed failed, skipping", "err", err)
				return
			}
			mu.Lock()
			embeddings[idx] = vec
			mu.Unlock()
		}(i, res.text)
	}
	wg.Wait()

	// Greedy suppression: keep the first (highest-score) chunk, drop any later
	// chunk whose embedding similarity to a kept chunk exceeds the threshold.
	dropped := make([]bool, len(results))
	for i := 0; i < len(results); i++ {
		if dropped[i] || embeddings[i] == nil {
			continue
		}
		for j := i + 1; j < len(results); j++ {
			if dropped[j] || embeddings[j] == nil {
				continue
			}
			sim := cosineSimilarity(embeddings[i], embeddings[j])
			if sim >= float32(threshold) {
				printer.ToolCall(printer.IconDone, "rag: dedup suppressed near-duplicate",
					"sim", fmt.Sprintf("%.3f", sim),
					"kept", preview(results[i].text, 40),
					"dropped", preview(results[j].text, 40),
				)
				dropped[j] = true
			}
		}
	}

	out := results[:0]
	for i, r := range results {
		if !dropped[i] {
			out = append(out, r)
		}
	}
	return out
}

// RagSave saves a text fragment to Qdrant. Returns true on success.
//
// The point ID is a deterministic UUID v5 derived from the content, making
// saves idempotent: re-indexing the same text overwrites the existing point
// instead of creating a duplicate entry.
func (r *RAGTools) RagSave(content string, prepend string, part int) bool {
	if prepend != "" {
		content = prepend + "\n" + content
	}

	id := contentUUID(content)

	printer.ToolCall(printer.IconSave, "rag_save", "id", id, "size", len(content), "part", part)

	vector, err := r.embed(content)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_save: embedding failed", "err", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	waitUpsert := true
	_, err = r.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.cfg.String("rag.qdrant.collection"),
		Wait:           &waitUpsert,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewID(id),
				Vectors: qdrant.NewVectors(vector...),
				Payload: qdrant.NewValueMap(map[string]any{
					payloadTextField: content,
				}),
			},
		},
	})
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_save: failed to upsert point", "id", id, "err", err)
		return false
	}

	return true
}

// RagSearch searches the knowledge base and returns relevant text fragments.
//
// When cfg.Int("rag.search.multi_query") > 0 the original query is expanded into
// multiple phrasings via the LLM; each phrasing is searched independently and
// results are merged by keeping the best (highest) score per unique chunk ID.

// When cfg.Float64("rag.search.dedup_threshold") > 0 near-duplicate chunks (cosine
// similarity above the threshold) are suppressed before returning results.
func (r *RAGTools) RagSearch(query string) string {
	printer.ToolCall(printer.IconSearch, "rad_search", "query", query)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// --- Step 1: build the list of queries to run ---
	queries := []string{query}
	if r.cfg.Int("rag.search.multi_query") > 0 {
		extras := r.expandQuery(ctx, query, r.cfg.Int("rag.search.multi_query"))
		queries = append(queries, extras...)
	}

	// --- Step 2: run all queries in parallel ---
	type queryResult struct {
		results []searchResult
		err     error
	}
	ch := make(chan queryResult, len(queries))

	for _, q := range queries {
		go func(q string) {
			res, err := r.searchOne(ctx, q)
			ch <- queryResult{res, err}
		}(q)
	}

	// --- Step 3: merge results, keeping best score per unique chunk ID ---
	bestByID := make(map[string]searchResult)
	for range queries {
		qr := <-ch
		if qr.err != nil {
			printer.Error("rag_search: query failed", "err", qr.err)
			continue
		}
		for _, res := range qr.results {
			if existing, ok := bestByID[res.id]; !ok || res.score > existing.score {
				bestByID[res.id] = res
			}
		}
	}

	if len(bestByID) == 0 {
		printer.ToolCall(printer.IconSearch, "rag_search: no results", "query", preview(query, 60))
		return ""
	}

	// Convert map to slice sorted by descending score.
	merged := make([]searchResult, 0, len(bestByID))
	for _, r := range bestByID {
		merged = append(merged, r)
	}
	sortByScore(merged)

	// Trim to configured limit (each sub-query can return up to Limit results,
	// so the merged set may be larger).
	if limit := r.cfg.Int("rag.search.limit"); limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	// --- Step 4: deduplicate near-identical chunks ---
	if r.cfg.Float64("rag.search.dedup_threshold") > 0 {
		merged = r.dedup(merged)
	}

	// --- Step 5: format output ---
	parts := make([]string, 0, len(merged))
	for _, res := range merged {
		parts = append(parts, res.text)
	}

	printer.ToolCall(printer.IconSearch, "rag_search: done", "query", preview(query, 60),
		"queries", len(queries), "results", len(parts))
	return strings.Join(parts, "\n\n---\n\n")
}

// sortByScore sorts results in descending order of score (insertion sort —
// result sets are small so allocation of a sort.Interface is not worth it).
func sortByScore(results []searchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

// preview returns the first n runes of s followed by "…" if s was truncated.
func preview(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// DefinitionSearch returns the OpenAI tool schema for rag_search.
func (r *RAGTools) DefinitionSearch() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "rag_search",
			Description: "Search the knowledge base for relevant information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Dispatch handles a tool call by name, parsing JSON args and returning the result as a string.
func (r *RAGTools) Dispatch(name string, args string) string {
	switch name {

	case "rag_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			printer.ToolCall(printer.IconError, "rag_search: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		result := r.RagSearch(params.Query)
		if result == "" {
			return "no relevant information found"
		}
		return result

	default:
		return "error: unknown tool " + name
	}
}

// StaticDefinitionSearch returns the OpenAI tool schema for rag_search without
// requiring an initialised RAGTools instance. Used by the plugin for lazy init.
func StaticDefinitionSearch() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "rag_search",
			Description: "Search the knowledge base for relevant information",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "Search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}
