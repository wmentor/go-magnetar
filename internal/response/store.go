package response

import (
	"errors"
	"sync"
)

// Store implements a circular buffer for storing responses.
// It is safe for concurrent use by multiple goroutines. A Store must not be
// copied.
type Store struct {
	mu     sync.RWMutex
	buffer []string
	cap    int
	head   int // index of the oldest element
	size   int // current number of elements
}

// ErrOutOfBounds is returned when index is out of valid range.
var ErrOutOfBounds = errors.New("index out of bounds")

// New creates a new Store with the given capacity.
func New(capacity int) *Store {
	if capacity <= 0 {
		capacity = 100 // default capacity
	}
	return &Store{
		buffer: make([]string, capacity),
		cap:    capacity,
		head:   0,
		size:   0,
	}
}

// Push adds a new response to the store.
// If the store is full, it overwrites the oldest element.
func (s *Store) Push(response string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.buffer[s.head] = response
	s.head = (s.head + 1) % s.cap
	if s.size < s.cap {
		s.size++
	}
}

// Last returns the response at the given index.
// index 0 = most recent, index 1 = second most recent, etc.
// Returns error if index is out of bounds.
func (s *Store) Last(idx int) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if idx < 0 {
		return "", ErrOutOfBounds
	}
	if idx >= s.size {
		return "", ErrOutOfBounds
	}

	// Calculate the actual index in the circular buffer
	// Most recent is at (head - 1) mod cap
	actualIdx := (s.head - 1 - idx + s.cap) % s.cap
	return s.buffer[actualIdx], nil
}

// Size returns the current number of elements in the store.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.size
}

// Capacity returns the maximum capacity of the store.
func (s *Store) Capacity() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cap
}
