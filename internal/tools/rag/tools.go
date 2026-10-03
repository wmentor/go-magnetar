package rag

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sashabaranov/go-openai"
)

// contentUUID derives a deterministic UUID v5 from the text content.
// Using uuid.NameSpaceDNS as the namespace gives a stable, collision-resistant
// identifier: identical content always maps to the same UUID, so upserting the
// same chunk twice is idempotent (Qdrant overwrites the existing point).
func ContentUUID(content string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(content)).String()
}

// Embed returns the embedding vector for the given text.
func Embed(ctx context.Context, embedClient *openai.Client, model string, text string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	resp, err := embedClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(model),
	})
	if err != nil {
		return nil, fmt.Errorf("rag: embedding failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("rag: empty embedding response")
	}

	return resp.Data[0].Embedding, nil
}

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
