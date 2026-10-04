package qdrant

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools/rag"
)

const (
	defaultGRPCPort = 6334
	defaultTimeout  = time.Second * 30

	payloadTextField = "text"
)

type Store struct {
	cfg          *config.Config
	embedClient  *openai.Client
	llmClient    *openai.Client // used for query expansion (multi-query)
	qdrantClient *qdrant.Client
}

// SearchResult holds one Qdrant result point with its vector for deduplication.
type SearchResult struct {
	ID     string
	Score  float32
	Text   string
	Vector []float32
}

func (s *Store) Name() string {
	return "qdrant"
}

// NewStore creates a new qdrant store instance.
func NewStore(cfg *config.Config) (*Store, error) {
	// Build OpenAI embedding client.
	embedCfg := openai.DefaultConfig(cfg.String("rag.llm.api_key"))
	embedCfg.BaseURL = cfg.String("rag.llm.base_url")
	embedClient := openai.NewClientWithConfig(embedCfg)

	// Build LLM client for query expansion (reuses main LLM config).
	llmCfg := openai.DefaultConfig(cfg.ProfileParamString("llm.api_key"))
	llmCfg.BaseURL = cfg.ProfileParamString("llm.base_url")
	llmClient := openai.NewClientWithConfig(llmCfg)

	// Parse connstr to extract host and port for gRPC.
	host, port, err := ParseConnStr(cfg.String("rag.qdrant.connstr"))
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

	rt := &Store{
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

// ensureCollection creates the Qdrant collection if it does not exist.
func (s *Store) ensureCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	exists, err := s.qdrantClient.CollectionExists(ctx, s.cfg.String("rag.qdrant.collection"))
	if err != nil {
		return fmt.Errorf("rag: failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	vectorSize := uint64(s.cfg.Int("rag.llm.vector_size"))
	err = s.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.cfg.String("rag.qdrant.collection"),
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("rag: failed to create collection %q: %w", s.cfg.String("rag.qdrant.collection"), err)
	}

	printer.Info("rag: collection created", "collection", s.cfg.String("rag.qdrant.collection"))
	return nil
}

// RagSave saves a text fragment to Qdrant. Returns true on success.
//
// The point ID is a deterministic UUID v5 derived from the content, making
// saves idempotent: re-indexing the same text overwrites the existing point
// instead of creating a duplicate entry.
func (s *Store) RagSave(content string, prepend string, part int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	if prepend != "" {
		content = prepend + "\n" + content
	}

	id := rag.ContentUUID(content)

	printer.ToolCall(printer.IconSave, "rag_save", "id", id, "size", len(content), "part", part)

	vector, err := rag.Embed(ctx, s.embedClient, s.cfg.String("rag.llm.model"), content)
	if err != nil {
		printer.ToolCall(printer.IconError, "rag_save: embedding failed", "err", err)
		return false
	}

	waitUpsert := true
	_, err = s.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.cfg.String("rag.qdrant.collection"),
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
func (s *Store) RagSearch(query string) string {
	printer.ToolCall(printer.IconSearch, "rag_search", "query", query)

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// --- Step 1: build the list of queries to run ---
	queries := []string{query}
	if s.cfg.Int("rag.search.multi_query") > 0 {
		extras := s.expandQuery(ctx, query, s.cfg.Int("rag.search.multi_query"))
		queries = append(queries, extras...)
	}

	// --- Step 2: run all queries in parallel ---
	type queryResult struct {
		results []SearchResult
		err     error
	}
	ch := make(chan queryResult, len(queries))

	for _, q := range queries {
		go func(q string) {
			res, err := s.searchOne(ctx, q)
			ch <- queryResult{res, err}
		}(q)
	}

	// --- Step 3: merge results, keeping best score per unique chunk ID ---
	bestByID := make(map[string]SearchResult)
	for range queries {
		qr := <-ch
		if qr.err != nil {
			printer.Error("rag_search: query failed", "err", qr.err)
			continue
		}
		for _, res := range qr.results {
			if existing, ok := bestByID[res.ID]; !ok || res.Score > existing.Score {
				bestByID[res.ID] = res
			}
		}
	}

	if len(bestByID) == 0 {
		printer.ToolCall(printer.IconSearch, "rag_search: no results", "query", Preview(query, 60))
		return ""
	}

	// Convert map to slice sorted by descending score.
	merged := make([]SearchResult, 0, len(bestByID))
	for _, r := range bestByID {
		merged = append(merged, r)
	}
	SortByScore(merged)

	// Trim to configured limit (each sub-query can return up to Limit results,
	// so the merged set may be larger).
	if limit := s.cfg.Int("rag.search.limit"); limit > 0 && len(merged) > limit {
		merged = merged[:limit]
	}

	// --- Step 4: deduplicate near-identical chunks ---
	if s.cfg.Float64("rag.search.dedup_threshold") > 0 {
		merged = s.dedup(merged)
	}

	// --- Step 5: format output ---
	parts := make([]string, 0, len(merged))
	for _, res := range merged {
		parts = append(parts, res.Text)
	}

	printer.ToolCall(printer.IconSearch, "rag_search: done", "query", Preview(query, 60),
		"queries", len(queries), "results", len(parts))
	return strings.Join(parts, "\n\n---\n\n")
}

// searchOne executes a single embedding + Qdrant query and returns raw results.
func (s *Store) searchOne(ctx context.Context, query string) ([]SearchResult, error) {
	vector, err := rag.Embed(ctx, s.embedClient, s.cfg.String("rag.llm.model"), query)
	if err != nil {
		return nil, err
	}

	qCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	limit := uint64(s.cfg.Int("rag.search.limit"))
	withPayload := true
	scoreThreshold := float32(s.cfg.Float64("rag.search.threshold"))
	points, err := s.qdrantClient.Query(qCtx, &qdrant.QueryPoints{
		CollectionName: s.cfg.String("rag.qdrant.collection"),
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		ScoreThreshold: &scoreThreshold,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		return nil, err
	}

	var out []SearchResult
	for _, p := range points {
		if payload := p.GetPayload(); payload != nil {
			if val, ok := payload[payloadTextField]; ok {
				if sv := val.GetStringValue(); sv != "" {
					out = append(out, SearchResult{
						ID:     p.GetId().GetUuid(),
						Score:  p.GetScore(),
						Text:   sv,
						Vector: vector, // reuse query vector for inter-result dedup
					})
				}
			}
		}
	}
	return out, nil
}

// expandQuery asks the LLM to produce n alternative phrasings of the query.
// Returns the reformulations (not including the original query).
// On any error it logs a warning and returns nil so the caller falls back to
// single-query mode gracefully.
func (s *Store) expandQuery(ctx context.Context, query string, n int) []string {
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

	resp, err := s.llmClient.CreateChatCompletion(reqCtx, openai.ChatCompletionRequest{
		Model: s.cfg.ProfileParamString("llm.model"),
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
	for line := range strings.SplitSeq(resp.Choices[0].Message.Content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	printer.ToolCall(printer.IconSearch, "rag_search: query expansion", "original", Preview(query, 60), "variants", len(result))
	return result
}

// dedup removes near-duplicate chunks from results.
// Two chunks are considered near-duplicates when their text embeddings have a
// cosine similarity above cfg.Float64("rag.search.dedup_threshold"). When a pair is found,
// the chunk with the lower score is dropped.
// Embeddings for each unique chunk are computed lazily and cached within the call.
func (s *Store) dedup(results []SearchResult) []SearchResult {
	threshold := s.cfg.Float64("rag.search.dedup_threshold")
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
			vec, err := rag.Embed(context.Background(), s.embedClient, s.cfg.String("rag.llm.model"), text)
			if err != nil {
				printer.ToolCall(printer.IconAlert, "rag: dedup embed failed, skipping", "err", err)
				return
			}
			mu.Lock()
			embeddings[idx] = vec
			mu.Unlock()
		}(i, res.Text)
	}
	wg.Wait()

	// Greedy suppression: keep the first (highest-score) chunk, drop any later
	// chunk whose embedding similarity to a kept chunk exceeds the threshold.
	dropped := make([]bool, len(results))
	for i := range results {
		if dropped[i] || embeddings[i] == nil {
			continue
		}
		for j := i + 1; j < len(results); j++ {
			if dropped[j] || embeddings[j] == nil {
				continue
			}
			sim := CosineSimilarity(embeddings[i], embeddings[j])
			if sim >= float32(threshold) {
				printer.ToolCall(printer.IconDone, "rag: dedup suppressed near-duplicate",
					"sim", fmt.Sprintf("%.3f", sim),
					"kept", Preview(results[i].Text, 40),
					"dropped", Preview(results[j].Text, 40),
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
