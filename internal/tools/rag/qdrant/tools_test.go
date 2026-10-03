package qdrant_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/rag/qdrant"
)

func TestParseConnStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		connStr  string
		wantHost string
		wantPort int
		wantErr  bool
	}{
		{
			name:     "default host and port",
			connStr:  "http://localhost:6333",
			wantHost: "localhost",
			wantPort: 6334,
		},
		{
			name:     "explicit gRPC port",
			connStr:  "http://localhost:6334",
			wantHost: "localhost",
			wantPort: 6334,
		},
		{
			name:     "custom host and REST port",
			connStr:  "http://qdrant.example.com:6333",
			wantHost: "qdrant.example.com",
			wantPort: 6334,
		},
		{
			name:     "custom host and gRPC port",
			connStr:  "http://qdrant.example.com:6334",
			wantHost: "qdrant.example.com",
			wantPort: 6334,
		},
		{
			name:     "different custom gRPC port",
			connStr:  "http://qdrant.example.com:9000",
			wantHost: "qdrant.example.com",
			wantPort: 9000,
		},
		{
			name:     "https protocol",
			connStr:  "https://qdrant.example.com:6333",
			wantHost: "qdrant.example.com",
			wantPort: 6334,
		},
		{
			name:     "localhost without port",
			connStr:  "http://localhost",
			wantHost: "localhost",
			wantPort: 6334,
		},
		{
			name:    "invalid port string",
			connStr: "http://localhost:abc",
			wantErr: true,
		},
		{
			name:     "invalid URL",
			connStr:  "not-a-url",
			wantHost: "localhost",
			wantPort: 6334,
		},
		{
			name:     "empty URL",
			connStr:  "",
			wantHost: "localhost",
			wantPort: 6334,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			host, port, err := qdrant.ParseConnStr(tt.connStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConnStr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if host != tt.wantHost {
				t.Errorf("parseConnStr() host = %q, want %q", host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("parseConnStr() port = %d, want %d", port, tt.wantPort)
			}
		})
	}
}

func TestSortByScore(t *testing.T) {
	t.Parallel()

	t.Run("empty slice", func(t *testing.T) {
		t.Parallel()
		results := []qdrant.SearchResult{}
		qdrant.SortByScore(results)
		if len(results) != 0 {
			t.Error("expected empty slice")
		}
	})

	t.Run("single element", func(t *testing.T) {
		t.Parallel()
		results := []qdrant.SearchResult{{Score: 0.5}}
		qdrant.SortByScore(results)
		if len(results) != 1 || results[0].Score != 0.5 {
			t.Error("single element should remain unchanged")
		}
	})

	t.Run("already sorted descending", func(t *testing.T) {
		t.Parallel()
		results := []qdrant.SearchResult{
			{Score: 0.9},
			{Score: 0.7},
			{Score: 0.5},
			{Score: 0.3},
		}
		qdrant.SortByScore(results)
		for i := 0; i < len(results)-1; i++ {
			if results[i].Score < results[i+1].Score {
				t.Errorf("results not sorted: %f < %f", results[i].Score, results[i+1].Score)
			}
		}
	})

	t.Run("unsorted ascending", func(t *testing.T) {
		t.Parallel()
		results := []qdrant.SearchResult{
			{Score: 0.1},
			{Score: 0.3},
			{Score: 0.2},
			{Score: 0.5},
			{Score: 0.4},
		}
		qdrant.SortByScore(results)
		for i := 0; i < len(results)-1; i++ {
			if results[i].Score < results[i+1].Score {
				t.Errorf("results not sorted: %f < %f", results[i].Score, results[i+1].Score)
			}
		}
		if results[0].Score != 0.5 || results[4].Score != 0.1 {
			t.Errorf("expected sorted [0.5, 0.4, 0.3, 0.2, 0.1], got %v", results)
		}
	})

	t.Run("equal scores", func(t *testing.T) {
		t.Parallel()
		results := []qdrant.SearchResult{
			{Score: 0.5},
			{Score: 0.5},
			{Score: 0.5},
		}
		qdrant.SortByScore(results)
		for i := range results {
			if results[i].Score != 0.5 {
				t.Errorf("all scores should be 0.5, got %f", results[i].Score)
			}
		}
	})
}

func TestPreview(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "short string",
			s:    "hello",
			n:    10,
			want: "hello",
		},
		{
			name: "exact length",
			s:    "hello",
			n:    5,
			want: "hello",
		},
		{
			name: "truncated",
			s:    "hello world",
			n:    5,
			want: "hello…",
		},
		{
			name: "unicode truncated",
			s:    "Привет мир",
			n:    6,
			want: "Привет…",
		},
		{
			name: "emoji",
			s:    "Hello 🌍 world",
			n:    7,
			want: "Hello 🌍…",
		},
		{
			name: "empty string",
			s:    "",
			n:    10,
			want: "",
		},
		{
			name: "zero length",
			s:    "hello",
			n:    0,
			want: "…",
		},
		{
			name: "single rune",
			s:    "a",
			n:    1,
			want: "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := qdrant.Preview(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("Preview() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCosineSimilarity(t *testing.T) {
	t.Parallel()

	t.Run("identical vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 0.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 1.0 {
			t.Errorf("identical vectors: similarity = %f, want 1.0", sim)
		}
	})

	t.Run("orthogonal vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 0.0}
		b := []float32{0.0, 1.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 0.0 {
			t.Errorf("orthogonal vectors: similarity = %f, want 0.0", sim)
		}
	})

	t.Run("opposite vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 1.0}
		b := []float32{-1.0, -1.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != -1.0 {
			t.Errorf("opposite vectors: similarity = %f, want -1.0", sim)
		}
	})

	t.Run("partially similar", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 0.5}
		b := []float32{1.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		expected := float32(1.0 / (2.0 * 1.0)) // 1.0 / sqrt(1.25) ≈ 0.894
		if sim < 0.89 || sim > 0.91 {
			t.Errorf("partially similar: similarity = %f, want ~%f", sim, expected)
		}
	})

	t.Run("different lengths", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 0.0}
		b := []float32{1.0, 0.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 0 {
			t.Errorf("different lengths: similarity = %f, want 0", sim)
		}
	})

	t.Run("zero vectors", func(t *testing.T) {
		t.Parallel()
		a := []float32{0.0, 0.0}
		b := []float32{0.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 0 {
			t.Errorf("zero vectors: similarity = %f, want 0", sim)
		}
	})

	t.Run("one zero vector", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 1.0}
		b := []float32{0.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 0 {
			t.Errorf("one zero vector: similarity = %f, want 0", sim)
		}
	})

	t.Run("unit vectors 3D", func(t *testing.T) {
		t.Parallel()
		a := []float32{1.0, 0.0, 0.0}
		b := []float32{0.0, 1.0, 0.0}
		sim := qdrant.CosineSimilarity(a, b)
		if sim != 0 {
			t.Errorf("unit vectors 3D: similarity = %f, want 0", sim)
		}
	})
}
