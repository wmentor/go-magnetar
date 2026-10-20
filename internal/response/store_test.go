package response

import (
	"testing"
)

func TestNewStore(t *testing.T) {
	t.Parallel()

	s := New(5)
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	if s.Capacity() != 5 {
		t.Errorf("expected capacity 5, got %d", s.Capacity())
	}
	if s.Size() != 0 {
		t.Errorf("expected size 0, got %d", s.Size())
	}
}

func TestPushAndLast(t *testing.T) {
	t.Parallel()

	s := New(3)

	// Push some elements
	s.Push("first")
	s.Push("second")
	s.Push("third")

	// Check sizes
	if s.Size() != 3 {
		t.Errorf("expected size 3, got %d", s.Size())
	}

	// Check Last with different indices
	val, err := s.Last(0)
	if err != nil {
		t.Fatalf("Last(0) failed: %v", err)
	}
	if val != "third" {
		t.Errorf("expected 'third', got '%s'", val)
	}

	val, err = s.Last(1)
	if err != nil {
		t.Fatalf("Last(1) failed: %v", err)
	}
	if val != "second" {
		t.Errorf("expected 'second', got '%s'", val)
	}

	val, err = s.Last(2)
	if err != nil {
		t.Fatalf("Last(2) failed: %v", err)
	}
	if val != "first" {
		t.Errorf("expected 'first', got '%s'", val)
	}
}

func TestPushOverwrite(t *testing.T) {
	t.Parallel()

	s := New(3)
	s.Push("first")
	s.Push("second")
	s.Push("third")
	s.Push("fourth") // should overwrite "first"

	if s.Size() != 3 {
		t.Errorf("expected size 3, got %d", s.Size())
	}

	// Check that "first" was overwritten (Last(2) should return the oldest element in buffer)
	val, err := s.Last(2)
	if err != nil {
		t.Fatalf("Last(2) failed: %v", err)
	}
	if val != "second" {
		t.Errorf("expected 'second', got '%s'", val)
	}

	// Check the order
	val, err = s.Last(0)
	if err != nil {
		t.Fatalf("Last(0) failed: %v", err)
	}
	if val != "fourth" {
		t.Errorf("expected 'fourth', got '%s'", val)
	}

	val, err = s.Last(1)
	if err != nil {
		t.Fatalf("Last(1) failed: %v", err)
	}
	if val != "third" {
		t.Errorf("expected 'third', got '%s'", val)
	}

	val, err = s.Last(2)
	if err != nil {
		t.Fatalf("Last(2) failed: %v", err)
	}
	if val != "second" {
		t.Errorf("expected 'second', got '%s'", val)
	}
}

func TestLastOutOfBounds(t *testing.T) {
	t.Parallel()

	s := New(3)
	s.Push("first")

	// Test negative index
	_, err := s.Last(-1)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds for negative index, got %v", err)
	}

	// Test index out of bounds
	_, err = s.Last(1)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds for index 1 with size 1, got %v", err)
	}

	// Push more elements
	s.Push("second")
	_, err = s.Last(2)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds for index 2 with size 2, got %v", err)
	}
}

func TestDefaultCapacity(t *testing.T) {
	t.Parallel()

	s := New(0)
	if s.Capacity() != 100 {
		t.Errorf("expected default capacity 100, got %d", s.Capacity())
	}

	s = New(-5)
	if s.Capacity() != 100 {
		t.Errorf("expected default capacity 100, got %d", s.Capacity())
	}
}

func TestCircularBufferWrapped(t *testing.T) {
	t.Parallel()

	s := New(3)

	// Fill the buffer
	s.Push("a")
	s.Push("b")
	s.Push("c")

	// Overwrite one element
	s.Push("d")

	// Now buffer contains: d, a, b (with head at index 1)
	// Last(0) = d, Last(1) = c (overwritten), Last(2) = b

	val, err := s.Last(0)
	if err != nil {
		t.Fatalf("Last(0) failed: %v", err)
	}
	if val != "d" {
		t.Errorf("expected 'd', got '%s'", val)
	}

	val, err = s.Last(1)
	if err != nil {
		t.Fatalf("Last(1) failed: %v", err)
	}
	if val != "c" {
		t.Errorf("expected 'c', got '%s'", val)
	}

	val, err = s.Last(2)
	if err != nil {
		t.Fatalf("Last(2) failed: %v", err)
	}
	if val != "b" {
		t.Errorf("expected 'b', got '%s'", val)
	}
}

func TestEmptyStore(t *testing.T) {
	t.Parallel()

	s := New(5)

	_, err := s.Last(0)
	if err != ErrOutOfBounds {
		t.Errorf("expected ErrOutOfBounds for empty store, got %v", err)
	}
}
