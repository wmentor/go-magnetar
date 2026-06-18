package chunk

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	maxDefault     = 512 // runes per chunk
	overlapDefault = 64  // runes of overlap
)

// Config holds the parameters for chunking.
type Config struct {
	// MaxSize is the maximum size of a chunk in runes (Unicode code points).
	MaxSize int
	// Overlap is the number of runes that overlap between adjacent chunks.
	// Overlap context is taken from the tail of the previous chunk and prepended
	// to the next chunk so that embeddings retain cross-boundary context.
	Overlap int
}

// DefaultConfig returns a config with sensible defaults tuned for RAG workloads.
// 512 runes fits comfortably inside a typical embedding model's 512-token limit
// (tokens ≈ runes for Latin text; for CJK it is still safe).
// 64-rune overlap (~12.5 %) preserves enough cross-boundary context without
// inflating storage significantly.
func DefaultConfig() Config {
	return Config{MaxSize: maxDefault, Overlap: overlapDefault}
}

// sanitize normalises line endings and cleans up trailing whitespace per line.
func sanitize(text string) string {
	// Normalise Windows/old-Mac line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

// isHeadingLine reports whether s is a Markdown ATX heading (# … ######).
func isHeadingLine(s string) bool {
	s = strings.TrimLeft(s, " \t")
	if !strings.HasPrefix(s, "#") {
		return false
	}
	i := 0
	for i < len(s) && s[i] == '#' {
		i++
	}
	// ATX heading: 1-6 '#' followed by a space/tab or end of string.
	return i >= 1 && i <= 6 && (i == len(s) || s[i] == ' ' || s[i] == '\t')
}

// splitParagraphs splits text into logical paragraphs.
//
// Paragraph boundaries are:
//  1. One or more blank lines (standard Markdown convention).
//  2. A Markdown ATX heading line (# … ######): each heading starts a new
//     paragraph so that section titles stay with the content that follows them.
//
// Blank lines between paragraphs are discarded. Lines within a paragraph are
// joined with a single newline.
func splitParagraphs(text string) []string {
	lines := strings.Split(text, "\n")

	var paragraphs []string
	var cur []string      // non-blank lines of the current paragraph
	inBlank := false      // true when we have seen ≥1 blank line since last content

	flushCur := func() {
		if len(cur) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(cur, "\n"))
		cur = nil
	}

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			// Blank line: if we have accumulated content, mark a boundary.
			if len(cur) > 0 {
				inBlank = true
			}
			continue
		}

		if isHeadingLine(line) {
			// Headings always start a fresh paragraph.
			flushCur()
			inBlank = false
			paragraphs = append(paragraphs, line)
			continue
		}

		// Regular non-blank line.
		if inBlank {
			// We crossed a blank-line boundary → flush the previous paragraph.
			flushCur()
			inBlank = false
		}
		cur = append(cur, line)
	}

	flushCur()
	return paragraphs
}

// runeLen returns the number of Unicode code points in s.
// It is O(n) but avoids the overhead of []rune conversion for the common case.
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

// runeSlice returns s[startRune:endRune] operating on rune indices.
func runeSlice(s string, startRune, endRune int) string {
	if startRune >= endRune {
		return ""
	}
	// Find byte offset of startRune.
	startByte := 0
	for i := 0; i < startRune; i++ {
		_, size := utf8.DecodeRuneInString(s[startByte:])
		startByte += size
	}
	// Find byte offset of endRune.
	endByte := startByte
	for i := startRune; i < endRune; i++ {
		r, size := utf8.DecodeRuneInString(s[endByte:])
		if r == utf8.RuneError && size == 0 {
			break
		}
		endByte += size
	}
	return s[startByte:endByte]
}

// findWordBoundaryBefore returns the rune index of the last word boundary at
// or before position pos within s (measured in runes). A word boundary is a
// transition between a space/punctuation character and a non-space character.
// If no boundary is found within a lookback window, pos is returned unchanged
// so the caller still makes progress.
func findWordBoundaryBefore(s string, pos int) int {
	const maxLookback = 80 // runes
	if pos <= 0 {
		return 0
	}
	runes := []rune(s)
	if pos >= len(runes) {
		pos = len(runes)
	}
	limit := pos - maxLookback
	if limit < 0 {
		limit = 0
	}
	// Walk backwards from pos looking for whitespace.
	for i := pos - 1; i >= limit; i-- {
		if unicode.IsSpace(runes[i]) {
			// Return the position after the whitespace run.
			j := i
			for j > limit && unicode.IsSpace(runes[j-1]) {
				j--
			}
			return j
		}
	}
	return pos
}

// overlapSuffix returns the last overlapRunes runes of s, snapped to a word
// boundary so that the overlap fragment starts at the beginning of a word.
// This avoids injecting a half-word at the start of the next chunk, which would
// confuse the embedding model.
func overlapSuffix(s string, overlapRunes int) string {
	n := runeLen(s)
	if overlapRunes <= 0 || n <= overlapRunes {
		return s
	}
	startRune := n - overlapRunes
	// Snap to a word boundary (look for nearest space before startRune+lookback).
	runes := []rune(s)
	// Walk forward from startRune to find the start of the next word.
	for startRune < n && unicode.IsSpace(runes[startRune]) {
		startRune++
	}
	return string(runes[startRune:])
}

// forceSplit splits a single paragraph that exceeds MaxSize into pieces,
// respecting word boundaries and UTF-8 rune boundaries.
//
// It operates entirely in rune space to avoid corrupting multibyte characters.
// Progress is guaranteed: each iteration advances start by at least 1 rune.
func forceSplit(para string, maxRunes int, overlapRunes int) []string {
	paraRunes := runeLen(para)
	if paraRunes <= maxRunes {
		return []string{para}
	}

	runes := []rune(para)
	total := len(runes)
	var chunks []string

	start := 0
	for start < total {
		end := start + maxRunes
		if end >= total {
			chunks = append(chunks, string(runes[start:]))
			break
		}

		// Try to snap the cut point back to a whitespace boundary so we do
		// not split mid-word. Walk leftward from end.
		cutEnd := end
		for cutEnd > start+1 && !unicode.IsSpace(runes[cutEnd-1]) {
			cutEnd--
		}
		if cutEnd <= start+1 {
			// No whitespace found in the range; hard-cut at maxRunes.
			cutEnd = end
		}

		// Trim any trailing whitespace from the emitted chunk.
		trimEnd := cutEnd
		for trimEnd > start && unicode.IsSpace(runes[trimEnd-1]) {
			trimEnd--
		}
		if trimEnd > start {
			chunks = append(chunks, string(runes[start:trimEnd]))
		}

		// Determine where the next window starts: skip whitespace after the cut.
		nextStart := cutEnd
		for nextStart < total && unicode.IsSpace(runes[nextStart]) {
			nextStart++
		}
		if nextStart >= total {
			break
		}

		// Apply overlap by stepping back from nextStart.
		if overlapRunes > 0 {
			overlapStart := nextStart - overlapRunes
			if overlapStart < start+1 {
				// Ensure we always advance at least 1 rune past the previous start.
				overlapStart = start + 1
			}
			// Snap overlapStart forward to a word start (avoid mid-word context).
			for overlapStart < nextStart && !unicode.IsSpace(runes[overlapStart-1]) && overlapStart > start+1 {
				overlapStart--
			}
			nextStart = overlapStart
		}

		// Safety: always advance by at least 1 rune to prevent infinite loops.
		if nextStart <= start {
			nextStart = start + 1
		}
		start = nextStart
	}

	return chunks
}

// Split divides text into overlapping chunks suitable for embedding and RAG
// retrieval.
//
// Strategy (in priority order):
//  1. Split on blank lines and Markdown headings (semantic paragraph boundaries).
//  2. Greedily pack consecutive paragraphs into a chunk up to MaxSize runes.
//  3. When a paragraph would overflow the current chunk:
//     a. Emit the current chunk.
//     b. Start the next chunk with an overlap suffix from the just-emitted chunk
//        (snapped to a word boundary) for cross-boundary context continuity.
//     c. If the paragraph itself is longer than MaxSize, force-split it and
//        apply the same overlap logic between the resulting sub-chunks.
//  4. All size accounting is in Unicode runes, not bytes, so multibyte
//     characters (CJK, emoji, Cyrillic, etc.) are handled correctly.
func Split(text string, cfg Config) []string {
	// Validate / normalise config.
	if cfg.MaxSize <= 0 {
		cfg = DefaultConfig()
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = 0
	}
	if cfg.Overlap >= cfg.MaxSize {
		cfg.Overlap = cfg.MaxSize / 4
	}

	text = sanitize(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return nil
	}

	var chunks []string

	// cur accumulates paragraphs for the current in-progress chunk.
	var cur []string
	curRunes := 0
	const sep = "\n\n"
	sepRunes := runeLen(sep)

	// emitCur flushes cur as a completed chunk and resets the accumulator.
	emitCur := func() {
		if len(cur) > 0 {
			chunks = append(chunks, strings.Join(cur, sep))
			cur = nil
			curRunes = 0
		}
	}

	// startOverlap prepends an overlap suffix from the last emitted chunk
	// (or last forceSplit piece) to the accumulator.
	startOverlap := func() {
		if len(chunks) == 0 || cfg.Overlap == 0 {
			return
		}
		tail := overlapSuffix(chunks[len(chunks)-1], cfg.Overlap)
		if tail != "" {
			cur = []string{tail}
			curRunes = runeLen(tail)
		}
	}

	for _, para := range paragraphs {
		paraRunes := runeLen(para)

		// Case 1: paragraph fits in the current chunk.
		addRunes := paraRunes
		if curRunes > 0 {
			addRunes += sepRunes
		}
		if curRunes+addRunes <= cfg.MaxSize {
			cur = append(cur, para)
			curRunes += addRunes
			continue
		}

		// Case 2: paragraph does NOT fit.
		emitCur()

		if paraRunes > cfg.MaxSize {
			// Case 2a: paragraph alone exceeds MaxSize → force-split.
			// Prepend any overlap from the previous chunk into the first sub-chunk.
			prefix := ""
			if len(chunks) > 0 && cfg.Overlap > 0 {
				prefix = overlapSuffix(chunks[len(chunks)-1], cfg.Overlap)
			}
			var paraText string
			if prefix != "" {
				paraText = prefix + " " + para
			} else {
				paraText = para
			}
			pieces := forceSplit(paraText, cfg.MaxSize, cfg.Overlap)
			chunks = append(chunks, pieces...)
			// Do NOT start cur with overlap yet; the next paragraph will do so.
			continue
		}

		// Case 2b: paragraph fits on its own — start a new chunk with overlap.
		startOverlap()
		addRunes = paraRunes
		if curRunes > 0 {
			addRunes += sepRunes
		}
		cur = append(cur, para)
		curRunes += addRunes
	}

	// Flush any remaining paragraphs.
	emitCur()

	return chunks
}
