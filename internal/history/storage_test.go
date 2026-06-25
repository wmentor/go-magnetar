package history_test

import (
	"testing"

	"github.com/wmentor/go-magnetar/internal/history"
)

const defaultHistoryLimit = 200

func TestStorage_EmptyFilename(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	if h.Records() == nil {
		t.Fatal("Records() returned nil")
	}

	if len(h.Records()) != 0 {
		t.Fatalf("expected empty records, got %d", len(h.Records()))
	}
}

func TestStorage_AddAndPrev(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	if prev := h.Prev(); prev != "third" {
		t.Fatalf("expected 'third', got '%s'", prev)
	}

	if prev := h.Prev(); prev != "second" {
		t.Fatalf("expected 'second', got '%s'", prev)
	}

	if prev := h.Prev(); prev != "first" {
		t.Fatalf("expected 'first', got '%s'", prev)
	}

	if prev := h.Prev(); prev != "" {
		t.Fatalf("expected empty string, got '%s'", prev)
	}
}

func TestStorage_PrevAtEnd(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	h.Add("first")
	h.Add("second")

	h.Prev()
	h.Prev()

	if prev := h.Prev(); prev != "" {
		t.Fatalf("expected empty string at end, got '%s'", prev)
	}
}

func TestStorage_AddAndNext(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	if next := h.Next(); next != "" {
		t.Fatalf("expected empty string at end, got '%s'", next)
	}

	h.Prev()
	h.Prev()

	if next := h.Next(); next != "third" {
		t.Fatalf("expected 'third', got '%s'", next)
	}

	if next := h.Next(); next != "" {
		t.Fatalf("expected empty string after last, got '%s'", next)
	}

	if next := h.Next(); next != "" {
		t.Fatalf("expected empty string after last, got '%s'", next)
	}
}

func TestStorage_NextAtEnd(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	h.Add("first")

	if next := h.Next(); next != "" {
		t.Fatalf("expected empty string at end, got '%s'", next)
	}
}

func TestStorage_RecordsAfterAdd(t *testing.T) {
	t.Parallel()
	h := history.New("", defaultHistoryLimit)

	h.Add("first")
	h.Add("second")
	h.Add("third")

	records := h.Records()
	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}

	if records[0] != "first" || records[1] != "second" || records[2] != "third" {
		t.Fatalf("unexpected records: %v", records)
	}
}
