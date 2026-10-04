package chromem

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// ParseDataDir parses the chromem data directory path, expanding ~ to home directory.
func ParseDataDir(dataDir string) (string, error) {
	if dataDir == "" {
		return "", fmt.Errorf("data_dir is required")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return strings.Replace(dataDir, "~", homeDir, 1), nil
}

// Preview returns the first n runes of s followed by "…" if s was truncated.
func Preview(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// CosineSimilarity computes the cosine similarity between two vectors.
func CosineSimilarity(a, b []float32) float32 {
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
