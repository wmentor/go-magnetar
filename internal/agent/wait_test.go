package agent

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitCanRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		parallelMode bool
		ctx          context.Context
		counter      int64
		wantErr      bool
		errContains  string
	}{
		{
			name:         "parallel mode without context cancellation",
			parallelMode: true,
			ctx:          context.Background(),
			counter:      0,
			wantErr:      false,
		},
		{
			name:         "sequential mode without contention (counter=0)",
			parallelMode: false,
			ctx:          context.Background(),
			counter:      0,
			wantErr:      false,
		},
		{
			name:         "context cancelled before call with contention",
			parallelMode: false,
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
			counter:     2,
			wantErr:     true,
			errContains: "context canceled",
		},
		{
			name:         "context with timeout before call with contention",
			parallelMode: false,
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				cancel()
				return ctx
			}(),
			counter:     2,
			wantErr:     true,
			errContains: "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			var counter atomic.Int64
			counter.Store(tt.counter)

			err := WaitCanRun(tt.ctx, &counter, tt.parallelMode)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errContains) {
						t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWaitCanRun_MultipleConcurrent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var counter atomic.Int64
	var wg sync.WaitGroup
	numGoroutines := 10
	results := make([]error, numGoroutines)

	// Run multiple goroutines concurrently in sequential mode
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = WaitCanRun(ctx, &counter, false)
			// Decrement counter after WaitCanRun returns
			counter.Add(-1)
		}(i)
	}

	// Wait for all goroutines to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines completed
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for goroutines to complete")
	}

	// Check that all goroutines succeeded
	for i, err := range results {
		if err != nil {
			t.Errorf("goroutine %d returned unexpected error: %v", i, err)
		}
	}
}

func TestWaitCanRun_ContextCancelDuringWait(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var counter atomic.Int64

	// Set counter to a value > 1 so that WaitCanRun will wait
	counter.Store(2)

	// Start a goroutine that will cancel the context after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Call WaitCanRun - it should eventually return with an error
	done := make(chan struct{})
	go func() {
		err := WaitCanRun(ctx, &counter, false)
		if err == nil {
			t.Errorf("expected error after context cancellation, got nil")
		}
		close(done)
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for context cancellation to be detected")
	}
}
