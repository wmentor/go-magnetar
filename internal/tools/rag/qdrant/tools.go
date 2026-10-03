package qdrant

import (
	"fmt"
	"math"
	"net/url"
	"strconv"
)

// ParseConnStr parses a URL like http://localhost:6333 and returns host and gRPC port (6334).
func ParseConnStr(connStr string) (string, int, error) {
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

// SortByScore sorts results in descending order of score (insertion sort —
// result sets are small so allocation of a sort.Interface is not worth it).
func SortByScore(results []SearchResult) {
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
}

// Preview returns the first n runes of s followed by "…" if s was truncated.
func Preview(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// CosineSimilarity computes the cosine similarity between two unit-normalised
// vectors. Qdrant returns cosine-distance results, so vectors from different
// queries may not be unit-normalised — we normalise here.
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
