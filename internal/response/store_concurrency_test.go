package response

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestStoreConcurrentAccess(t *testing.T) {
	t.Parallel()

	s := New(8)

	// Seed the store so every read below has to succeed: writers only ever add
	// responses, so the store can never become empty again.
	s.Push("seed")

	const (
		writers = 8
		readers = 4
		rounds  = 500
	)

	var wg sync.WaitGroup

	start := make(chan struct{})

	for w := range writers {
		wg.Go(func() {
			<-start

			for i := range rounds {
				s.Push(fmt.Sprintf("w%d-%d", w, i))
			}
		})
	}

	for range readers {
		wg.Go(func() {
			<-start

			for range rounds {
				val, err := s.Last(0)
				if err != nil {
					t.Errorf("unexpected error from Last(0): %v", err)
				} else if val == "" {
					t.Error("last response is empty")
				}

				if size := s.Size(); size < 0 || size > s.Capacity() {
					t.Errorf("size %d out of range 0..%d", size, s.Capacity())
				}
			}
		})
	}

	close(start)

	wg.Wait()

	if got := s.Size(); got != s.Capacity() {
		t.Errorf("expected a full store of %d, got %d", s.Capacity(), got)
	}

	if _, err := s.Last(s.Capacity()); !errors.Is(err, ErrOutOfBounds) {
		t.Errorf("expected ErrOutOfBounds past capacity, got %v", err)
	}
}
