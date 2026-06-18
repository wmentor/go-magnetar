package chunk

import "strings"

const (
	maxDefault     = 450  // characters per chunk
	overlapDefault = 50  // characters of overlap
)

// Config holds the parameters for chunking.
type Config struct {
	// MaxSize is the maximum size of a chunk in characters.
	MaxSize int
	// Overlap is the number of characters that overlap between adjacent chunks.
	Overlap int
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() Config {
	return Config{MaxSize: maxDefault, Overlap: overlapDefault}
}

// splitParagraphs splits text on one or more blank lines.
// Each non-blank line within a paragraph is joined with a single newline.
func splitParagraphs(text string) []string {
	raw := strings.Split(text, "\n\n")
	var result []string
	for _, block := range raw {
		lines := strings.Split(block, "\n")
		var sb strings.Builder
		for _, line := range lines {
			s := strings.TrimSpace(line)
			if s != "" {
				if sb.Len() > 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(s)
			}
		}
		t := strings.TrimSpace(sb.String())
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}

// Split divides text into overlapping chunks.
// It tries to split at paragraph boundaries first (separated by blank lines).
// If a single paragraph exceeds max size, it is split by lines with overlap.
func Split(text string, cfg Config) []string {
	if cfg.MaxSize <= 0 {
		cfg = DefaultConfig()
	}
	if cfg.Overlap < 0 {
		cfg.Overlap = 0
	}
	if cfg.Overlap >= cfg.MaxSize {
		cfg.Overlap = cfg.MaxSize / 4
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return nil
	}

	var chunks []string
	var cur []string
	curSize := 0

	for i, para := range paragraphs {
		paraLen := len(para)
		seps := 2 // "\n\n" between paragraphs

		// Would this paragraph exceed the max size?
		if curSize > 0 && curSize+seps+paraLen > cfg.MaxSize {
			chunks = append(chunks, strings.Join(cur, "\n\n"))
			cur = nil
			curSize = 0

			if paraLen > cfg.MaxSize {
				// Force-split this oversized paragraph.
				chunks = append(chunks, forceSplit(para, cfg.MaxSize, cfg.Overlap)...)
				continue
			}

			// Start a new chunk with an overlap from the previous chunk.
			if i > 0 {
				prevText := chunks[len(chunks)-1]
				if len(prevText) > cfg.Overlap {
					overlap := prevText[len(prevText)-cfg.Overlap:]
					cur = []string{overlap, para}
					curSize = len(overlap) + seps + paraLen
				} else {
					cur = []string{para}
					curSize = paraLen
				}
				continue
			}

			cur = []string{para}
			curSize = paraLen
			continue
		}

		cur = append(cur, para)
		if curSize > 0 {
			curSize += seps + paraLen
		} else {
			curSize = paraLen
		}

		// Emit if we've reached or exceeded max size.
		if i == len(paragraphs)-1 || (len(chunks) > 0 && curSize >= cfg.MaxSize) {
			if len(cur) > 0 {
				chunks = append(chunks, strings.Join(cur, "\n\n"))
			}
			cur = nil
			curSize = 0
		}
	}

	return chunks
}

// forceSplit splits para into pieces respecting overlap between pieces.
func forceSplit(para string, max int, overlap int) []string {
	if len(para) <= max {
		return []string{para}
	}

	var chunks []string
	for start := 0; start < len(para); {
		end := start + max
		if end >= len(para) {
			chunks = append(chunks, para[start:])
			break
		}
		chunks = append(chunks, para[start:end])
		start = end - overlap
	}

	return chunks
}
