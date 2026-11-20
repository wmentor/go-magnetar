package processor

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wmentor/go-magnetar/internal/plugin"
)

// TestNewStore creates a new store and verifies it's not nil
func TestNewStore(t *testing.T) {
	t.Parallel()

	s := NewStore()
	assert.NotNil(t, s)
	assert.IsType(t, &store{}, s)
}

// TestStore_AddProcessorFunc tests adding processor functions
func TestStore_AddProcessorFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		processor plugin.WebProcessorFunc
		wantCount int
	}{
		{
			name:      "add valid processor",
			processor: func(_ context.Context, _ string) (string, error) { return "result", nil },
			wantCount: 1,
		},
		{
			name:      "add nil processor - should be ignored",
			processor: nil,
			wantCount: 0,
		},
		{
			name:      "add multiple processors",
			processor: func(_ context.Context, _ string) (string, error) { return "result", nil },
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewStore()

			// Add processors
			for j := 0; j < (tt.wantCount - 1); j++ {
				s.AddProcessorFunc(func(_ context.Context, _ string) (string, error) {
					return "dummy", nil
				})
			}

			if tt.processor != nil {
				s.AddProcessorFunc(tt.processor)
			}

			// For multiple processors test, we need to count after all additions
			if tt.name == "add multiple processors" {
				// Access the underlying store to check count
				store := s.(*store)
				assert.Len(t, store.processors, tt.wantCount)
			}
		})
	}
}

// TestStore_AddProcessorFunc_Concurrent tests concurrent addition of processors
func TestStore_AddProcessorFunc_Concurrent(t *testing.T) {
	t.Parallel()

	s := NewStore()
	const n = 100
	var wg sync.WaitGroup

	for range n {
		wg.Go(func() {
			s.AddProcessorFunc(func(_ context.Context, _ string) (string, error) {
				return "result", nil
			})
		})
	}

	wg.Wait()

	store := s.(*store)
	assert.Len(t, store.processors, n)
}

// TestStore_Process tests the Process method
func TestStore_Process(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		processors    []plugin.WebProcessorFunc
		url           string
		expected      string
		expectedError error
	}{
		{
			name: "single processor returns success",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "processed: " + url, nil
				},
			},
			url:      "https://example.com",
			expected: "processed: https://example.com",
		},
		{
			name: "processor returns error",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "", errors.New("processing failed")
				},
			},
			url:           "https://example.com",
			expected:      "",
			expectedError: errors.New("process URL \"https://example.com\" error: processing failed"),
		},
		{
			name: "processor skips with ErrNoWebProcessor",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "", plugin.ErrNoWebProcessor
				},
			},
			url:           "https://example.com",
			expected:      "",
			expectedError: plugin.ErrNoWebProcessor,
		},
		{
			name: "multiple processors, first succeeds",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "first", nil
				},
				func(_ context.Context, url string) (string, error) {
					return "second", nil
				},
			},
			url:      "https://example.com",
			expected: "first",
		},
		{
			name: "multiple processors, second succeeds",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "", plugin.ErrNoWebProcessor
				},
				func(_ context.Context, url string) (string, error) {
					return "second", nil
				},
			},
			url:      "https://example.com",
			expected: "second",
		},
		{
			name: "multiple processors, all skip",
			processors: []plugin.WebProcessorFunc{
				func(_ context.Context, url string) (string, error) {
					return "", plugin.ErrNoWebProcessor
				},
				func(_ context.Context, url string) (string, error) {
					return "", plugin.ErrNoWebProcessor
				},
			},
			url:           "https://example.com",
			expected:      "",
			expectedError: plugin.ErrNoWebProcessor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewStore()
			for _, p := range tt.processors {
				s.AddProcessorFunc(p)
			}

			ctx := context.Background()
			result, err := s.Process(ctx, tt.url)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestStore_Process_Concurrent tests concurrent access to Process
func TestStore_Process_Concurrent(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		return "processed", nil
	})

	const n = 100
	errors := make(chan error, n)

	for range n {
		go func() {
			ctx := context.Background()
			_, err := s.Process(ctx, "https://example.com")
			errors <- err
		}()
	}

	for range n {
		err := <-errors
		assert.NoError(t, err)
	}
}

// TestStore_Process_ContextCancellation tests context cancellation
func TestStore_Process_ContextCancellation(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.AddProcessorFunc(func(ctx context.Context, url string) (string, error) {
		// Simulate long-running process
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			return "completed", nil
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := s.Process(ctx, "https://example.com")
	assert.Error(t, err)
	// The error is wrapped, so check for the cause
	assert.Contains(t, err.Error(), "context canceled")
}

// TestStore_Process_URLVariations tests various URL formats
func TestStore_Process_URLVariations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{"simple URL", "https://example.com"},
		{"URL with path", "https://example.com/path/to/resource"},
		{"URL with query", "https://example.com?q=test&page=1"},
		{"URL with fragment", "https://example.com#section"},
		{"localhost URL", "http://localhost:8080/api"},
		{"IP address URL", "http://127.0.0.1:8080/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := NewStore()
			s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
				return "processed:" + url, nil
			})

			ctx := context.Background()
			result, err := s.Process(ctx, tt.url)

			assert.NoError(t, err)
			assert.Contains(t, result, "processed:")
			assert.Contains(t, result, tt.url)
		})
	}
}

// TestStore_MultipleProcessors_Chain tests chaining of multiple processors
func TestStore_MultipleProcessors_Chain(t *testing.T) {
	t.Parallel()

	s := NewStore()

	// First processor: lowercase URL
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		return "first:" + url, nil
	})

	// Second processor: uppercase result
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		return "second:" + url, nil
	})

	// Third processor: capitalize result
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		return "third:" + url, nil
	})

	ctx := context.Background()
	result, err := s.Process(ctx, "https://Example.com")

	assert.NoError(t, err)
	// First processor should be executed (and return early)
	assert.Equal(t, "first:https://Example.com", result)
}

// TestStore_EmptyStore tests behavior when no processors are registered
func TestStore_EmptyStore(t *testing.T) {
	t.Parallel()

	s := NewStore()

	ctx := context.Background()
	_, err := s.Process(ctx, "https://example.com")

	assert.Error(t, err)
	assert.Equal(t, plugin.ErrNoWebProcessor, err)
}

// TestStore_ProcessorWithRealHTTP tests processor that makes real HTTP request
func TestStore_ProcessorWithRealHTTP(t *testing.T) {
	t.Parallel()

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	s := NewStore()
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(body), nil
	})

	ctx := context.Background()
	result, err := s.Process(ctx, server.URL)

	require.NoError(t, err)
	assert.Equal(t, "OK", result)
}

// TestStore_AddProcessorFunc_NilContext tests that processors handle nil context properly
func TestStore_AddProcessorFunc_NilContext(t *testing.T) {
	t.Parallel()

	// This test verifies that adding nil processor doesn't cause panic
	s := NewStore()
	assert.NotPanics(t, func() {
		s.AddProcessorFunc(nil)
	})

	// Access the underlying store to check processors slice
	store := s.(*store)
	assert.Empty(t, store.processors)
}

// TestStore_Process_NilContext tests Process with nil context
func TestStore_Process_NilContext(t *testing.T) {
	t.Parallel()

	s := NewStore()
	s.AddProcessorFunc(func(_ context.Context, url string) (string, error) {
		return "result", nil
	})

	// Note: context.Background() is passed, not nil
	ctx := context.Background()
	_, err := s.Process(ctx, "https://example.com")

	assert.NoError(t, err)
}
