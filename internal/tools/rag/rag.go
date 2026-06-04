package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"

	"github.com/wmentor/go-magnetar/internal/config"
)

const (
	payloadTextField = "text"
	searchLimit      = 5
	defaultGRPCPort  = 6334
)

// RAGTools provides RAG operations as LLM tools.
type RAGTools struct {
	cfg          *config.Config
	embedClient  *openai.Client
	qdrantClient *qdrant.Client
}

// New creates a new RAGTools instance, connects to Qdrant and ensures the collection exists.
func New(cfg *config.Config) (*RAGTools, error) {
	// Build OpenAI embedding client.
	embedCfg := openai.DefaultConfig(cfg.RAG.LLM.APIKey)
	embedCfg.BaseURL = cfg.RAG.LLM.BaseURL
	embedClient := openai.NewClientWithConfig(embedCfg)

	// Parse connstr to extract host and port for gRPC.
	host, port, err := parseConnStr(cfg.RAG.Qdrant.ConnStr)
	if err != nil {
		return nil, fmt.Errorf("rag: invalid qdrant connstr %q: %w", cfg.RAG.Qdrant.ConnStr, err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	exists, err := r.qdrantClient.CollectionExists(ctx, r.cfg.RAG.Qdrant.Collection)
	if err != nil {
		return fmt.Errorf("rag: failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	vectorSize := uint64(r.cfg.RAG.LLM.VectorSize)
	err = r.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: r.cfg.RAG.Qdrant.Collection,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		return fmt.Errorf("rag: failed to create collection %q: %w", r.cfg.RAG.Qdrant.Collection, err)
	}

	slog.Info("rag: collection created", "collection", r.cfg.RAG.Qdrant.Collection)
	return nil
}

// embed returns the embedding vector for the given text.
func (r *RAGTools) embed(text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := r.embedClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(r.cfg.RAG.LLM.Model),
	})
	if err != nil {
		return nil, fmt.Errorf("rag: embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("rag: empty embedding response")
	}

	return resp.Data[0].Embedding, nil
}

// RagSave saves a text fragment to Qdrant. Returns true on success.
func (r *RAGTools) RagSave(content string) bool {
	id := uuid.NewString()

	vector, err := r.embed(content)
	if err != nil {
		slog.Error("rag_save: embedding failed", "err", err)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	waitUpsert := true
	_, err = r.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: r.cfg.RAG.Qdrant.Collection,
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
		slog.Error("rag_save: failed to upsert point", "id", id, "err", err)
		return false
	}

	return true
}

// RagSearch searches the knowledge base and returns relevant text fragments.
func (r *RAGTools) RagSearch(query string) string {
	vector, err := r.embed(query)
	if err != nil {
		slog.Error("rag_search: embedding failed", "err", err)
		return ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	limit := uint64(searchLimit)
	withPayload := true
	results, err := r.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: r.cfg.RAG.Qdrant.Collection,
		Query:          qdrant.NewQuery(vector...),
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: withPayload}},
	})
	if err != nil {
		slog.Error("rag_search: query failed", "err", err)
		return ""
	}

	var parts []string
	for _, point := range results {
		if payload := point.GetPayload(); payload != nil {
			if val, ok := payload[payloadTextField]; ok {
				if sv := val.GetStringValue(); sv != "" {
					parts = append(parts, sv)
				}
			}
		}
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// DefinitionSave returns the OpenAI tool schema for rag_save.
func (r *RAGTools) DefinitionSave() openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "rag_save",
			Description: "Save a text fragment to the knowledge base",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "Text fragment to save",
					},
				},
				"required": []string{"content"},
			},
		},
	}
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
	case "rag_save":
		var params struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("rag_save: failed to parse args", "args", args, "err", err)
			return "error: failed to parse arguments"
		}
		ok := r.RagSave(params.Content)
		if !ok {
			return "error: failed to save to RAG"
		}
		return "saved successfully"

	case "rag_search":
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			slog.Error("rag_search: failed to parse args", "args", args, "err", err)
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
