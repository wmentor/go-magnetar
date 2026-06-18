package chunk

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// ---------- helpers ----------

func cfg(maxSize, overlap int) Config {
	return Config{MaxSize: maxSize, Overlap: overlap}
}

// allRunesValid verifies that every chunk is valid UTF-8.
func allRunesValid(t *testing.T, chunks []string) {
	t.Helper()
	for i, c := range chunks {
		if !utf8.ValidString(c) {
			t.Errorf("chunk %d is not valid UTF-8: %q", i, c)
		}
	}
}

// noChunkExceedsMax checks that every chunk is at most maxRunes runes long.
func noChunkExceedsMax(t *testing.T, chunks []string, maxRunes int) {
	t.Helper()
	for i, c := range chunks {
		n := runeLen(c)
		if n > maxRunes {
			t.Errorf("chunk %d has %d runes, want ≤ %d; value: %q", i, n, maxRunes, c)
		}
	}
}

// ---------- splitParagraphs ----------

func TestSplitParagraphsBasic(t *testing.T) {
	in := "hello world\n\ngoodbye world"
	got := splitParagraphs(in)
	if len(got) != 2 {
		t.Fatalf("want 2 paragraphs, got %d: %v", len(got), got)
	}
	if got[0] != "hello world" {
		t.Errorf("para[0] = %q, want %q", got[0], "hello world")
	}
	if got[1] != "goodbye world" {
		t.Errorf("para[1] = %q, want %q", got[1], "goodbye world")
	}
}

func TestSplitParagraphsMultipleBlankLines(t *testing.T) {
	in := "a\n\n\n\nb"
	got := splitParagraphs(in)
	if len(got) != 2 {
		t.Fatalf("want 2 paragraphs, got %d: %v", len(got), got)
	}
}

func TestSplitParagraphsMarkdownHeadings(t *testing.T) {
	in := "# Title\n\nsome text\n\n## Section\n\nmore text"
	got := splitParagraphs(in)
	// Expected: "# Title", "some text", "## Section", "more text"
	if len(got) != 4 {
		t.Fatalf("want 4 paragraphs, got %d: %v", len(got), got)
	}
	if got[0] != "# Title" {
		t.Errorf("para[0] = %q", got[0])
	}
	if got[2] != "## Section" {
		t.Errorf("para[2] = %q", got[2])
	}
}

func TestSplitParagraphsWindowsLineEndings(t *testing.T) {
	// sanitize is called inside Split, not splitParagraphs, so we must test via Split.
	in := "line1\r\nline2\r\n\r\nline3"
	chunks := Split(in, cfg(512, 0))
	if len(chunks) == 0 {
		t.Fatal("got no chunks")
	}
	// The two lines should appear; CRLF should be gone.
	joined := strings.Join(chunks, " ")
	if strings.Contains(joined, "\r") {
		t.Errorf("carriage return survived sanitization: %q", joined)
	}
	if !strings.Contains(joined, "line1") || !strings.Contains(joined, "line3") {
		t.Errorf("content lost: %q", joined)
	}
}

// ---------- Split — basic behaviour ----------

func TestSplitEmptyInput(t *testing.T) {
	if got := Split("", cfg(100, 10)); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
	if got := Split("   \n\n  ", cfg(100, 10)); got != nil {
		t.Errorf("expected nil for blank input, got %v", got)
	}
}

func TestSplitSingleShortText(t *testing.T) {
	in := "Hello, world!"
	chunks := Split(in, cfg(100, 10))
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != in {
		t.Errorf("chunk[0] = %q, want %q", chunks[0], in)
	}
}

func TestSplitNoChunkExceedsMax(t *testing.T) {
	// Long repeating text that will definitely be split.
	word := "lorem ipsum dolor sit amet, consectetur adipiscing elit. "
	in := strings.Repeat(word, 30)
	maxSize := 200
	chunks := Split(in, cfg(maxSize, 20))
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	noChunkExceedsMax(t, chunks, maxSize)
	allRunesValid(t, chunks)
}

func TestSplitContentPreserved(t *testing.T) {
	// All original words should appear in at least one chunk.
	words := []string{"alpha", "beta", "gamma", "delta", "epsilon", "zeta"}
	in := strings.Join(words, "\n\n")
	chunks := Split(in, cfg(20, 5))
	joined := strings.Join(chunks, " ")
	for _, w := range words {
		if !strings.Contains(joined, w) {
			t.Errorf("word %q not found in any chunk", w)
		}
	}
}

// ---------- Split — last chunk is always emitted ----------

func TestSplitLastChunkEmitted(t *testing.T) {
	// Regression: old code had a bug where the last chunk was dropped when
	// curSize < MaxSize and it was the only chunk (len(chunks)==0).
	in := "short paragraph"
	chunks := Split(in, cfg(1000, 50))
	if len(chunks) != 1 {
		t.Fatalf("want 1 chunk, got %d", len(chunks))
	}
}

func TestSplitLastParagraphNotDropped(t *testing.T) {
	// Several paragraphs that each fit; none should be dropped.
	paras := []string{"para one", "para two", "para three", "para four"}
	in := strings.Join(paras, "\n\n")
	chunks := Split(in, cfg(512, 0))
	joined := strings.Join(chunks, " ")
	for _, p := range paras {
		if !strings.Contains(joined, p) {
			t.Errorf("paragraph %q not found in any chunk", p)
		}
	}
}

// ---------- Split — overlap ----------

func TestSplitOverlapPresent(t *testing.T) {
	// Two paragraphs that are each big enough to force a new chunk.
	// The second chunk should contain a suffix of the first.
	para1 := strings.Repeat("a ", 60) // 120 runes
	para2 := strings.Repeat("b ", 60) // 120 runes
	in := para1 + "\n\n" + para2
	chunks := Split(in, cfg(130, 20))
	if len(chunks) < 2 {
		t.Fatalf("expected ≥2 chunks, got %d", len(chunks))
	}
	// chunk[1] must contain some 'a' characters from the overlap.
	if !strings.Contains(chunks[1], "a") {
		t.Errorf("overlap from first chunk not found in second chunk: %q", chunks[1])
	}
}

func TestSplitZeroOverlapNoLeakage(t *testing.T) {
	para1 := strings.Repeat("x ", 60)
	para2 := strings.Repeat("y ", 60)
	in := para1 + "\n\n" + para2
	chunks := Split(in, cfg(130, 0))
	if len(chunks) < 2 {
		t.Fatalf("expected ≥2 chunks, got %d", len(chunks))
	}
	// With zero overlap chunk[1] should not contain 'x'.
	if strings.Contains(chunks[1], "x") {
		t.Errorf("unexpected overlap leakage with Overlap=0: %q", chunks[1])
	}
}

// ---------- Split — UTF-8 / multibyte ----------

func TestSplitUTF8Cyrillic(t *testing.T) {
	// Each Cyrillic char is 2 bytes. Splitting by bytes would break them.
	in := strings.Repeat("Привет мир! ", 40)
	maxSize := 50
	chunks := Split(in, cfg(maxSize, 10))
	noChunkExceedsMax(t, chunks, maxSize)
	allRunesValid(t, chunks)
}

func TestSplitUTF8CJK(t *testing.T) {
	// CJK characters: 3 bytes each.
	in := strings.Repeat("你好世界！ ", 40)
	maxSize := 30
	chunks := Split(in, cfg(maxSize, 5))
	noChunkExceedsMax(t, chunks, maxSize)
	allRunesValid(t, chunks)
}

func TestSplitUTF8Emoji(t *testing.T) {
	// Emoji: 4 bytes each.
	in := strings.Repeat("🎉🚀🌍 go-magnetar rocks! ", 20)
	maxSize := 40
	chunks := Split(in, cfg(maxSize, 8))
	noChunkExceedsMax(t, chunks, maxSize)
	allRunesValid(t, chunks)
}

// ---------- forceSplit directly ----------

func TestForceSplitUTF8(t *testing.T) {
	// A long Cyrillic string with no spaces (worst case for word boundary search).
	in := strings.Repeat("Ж", 300)
	maxSize := 100
	pieces := forceSplit(in, maxSize, 10)
	for i, p := range pieces {
		if !utf8.ValidString(p) {
			t.Errorf("piece %d is not valid UTF-8", i)
		}
		n := runeLen(p)
		if n > maxSize {
			t.Errorf("piece %d has %d runes, want ≤ %d", i, n, maxSize)
		}
	}
}

func TestForceSplitShortString(t *testing.T) {
	in := "hello"
	pieces := forceSplit(in, 100, 10)
	if len(pieces) != 1 || pieces[0] != in {
		t.Errorf("short string should not be split: %v", pieces)
	}
}

// ---------- Config validation ----------

func TestSplitInvalidConfigFallsToDefault(t *testing.T) {
	in := "some text here"
	// MaxSize <= 0 should trigger DefaultConfig.
	chunks := Split(in, Config{MaxSize: 0, Overlap: 0})
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk with default config")
	}
}

func TestSplitOverlapGeMaxSizeClamped(t *testing.T) {
	// overlap >= maxSize must be clamped to maxSize/4.
	in := strings.Repeat("word ", 200)
	chunks := Split(in, Config{MaxSize: 100, Overlap: 100})
	noChunkExceedsMax(t, chunks, 100)
}

// ---------- DefaultConfig ----------

func TestDefaultConfig(t *testing.T) {
	dc := DefaultConfig()
	if dc.MaxSize != maxDefault {
		t.Errorf("MaxSize = %d, want %d", dc.MaxSize, maxDefault)
	}
	if dc.Overlap != overlapDefault {
		t.Errorf("Overlap = %d, want %d", dc.Overlap, overlapDefault)
	}
}

// ---------- isHeadingLine ----------

func TestIsHeadingLine(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"# Title", true},
		{"## Sub", true},
		{"###### Deep", true},
		{"####### Too deep", false},
		{"#NoSpace", false},
		{"Not a heading", false},
		{"", false},
		{"  # Indented heading", true},
	}
	for _, tc := range cases {
		if got := isHeadingLine(tc.s); got != tc.want {
			t.Errorf("isHeadingLine(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}
