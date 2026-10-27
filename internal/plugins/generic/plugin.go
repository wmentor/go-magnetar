package genericplugin

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wmentor/go-magnetar/internal/codec"
	"github.com/wmentor/go-magnetar/internal/common"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func init() {
	plugin.Register("generic", &Plugin{})
}

// Plugin wraps the generic file tools and exposes file_list,
// file_write, and system_grep (if grep is available) as LLM tools.
// The GenericTools instance is created lazily on first use so that the
// agent's working-directory Root (set via plugin.SetRoot) is available.
type Plugin struct {
	state *plugin.State
}

func (p *Plugin) Init(s *plugin.State, hub plugin.Hub) error {
	p.state = s
	p.registerPreprocessor(hub)
	return nil
}

func (p *Plugin) registerPreprocessor(hub plugin.Hub) {
	hub.RegisterPreprocessor(func(ctx context.Context, text string) (string, error) {
		if home, err := os.UserHomeDir(); err == nil {
			text = strings.ReplaceAll(text, "{{home}}", home)
		}

		text = strings.ReplaceAll(text, "{{uuid}}", uuid.New().String())

		text = strings.ReplaceAll(text, "{{date}}", time.Now().Format("2006-01-02"))
		text = strings.ReplaceAll(text, "{{now}}", time.Now().Format("2006-01-02 15:04:05"))

		text = processFiles(text)
		text = processEnvVars(text)

		return text, nil
	})
}

func processFiles(text string) string {
	// Process {{file:filename}} patterns
	// Pattern: {{file:...}} where ... can be absolute path or ~/path
	repl := func(match string) string {
		// Extract filename from {{file:filename}}
		start := strings.Index(match, "{{file:")
		if start == -1 {
			return match
		}
		start += len("{{file:")
		end := strings.Index(match[start:], "}}")
		if end == -1 {
			return match
		}
		end += start

		filename := common.ExpandHome(match[start:end])

		// Read file and return its content
		content := readFile(filename)
		if content == "" {
			return match // Return original if file couldn't be read
		}
		return content
	}

	// Find and replace all {{file:...}} patterns
	return replaceAllPatterns(text, "{{file:", "}}", repl)
}

func processEnvVars(text string) string {
	// Process {{env:VARIABLE}} patterns
	repl := func(match string) string {
		start := strings.Index(match, "{{env:")
		if start == -1 {
			return match
		}
		start += len("{{env:")
		end := strings.Index(match[start:], "}}")
		if end == -1 {
			return match
		}
		end += start

		variable := match[start:end]
		value := os.Getenv(variable)

		return value
	}

	// Find and replace all {{env:...}} patterns
	return replaceAllPatterns(text, "{{env:", "}}", repl)
}

func replaceAllPatterns(s, prefix, suffix string, repl func(string) string) string {
	result := s
	searchFrom := 0
	for {
		start := strings.Index(result[searchFrom:], prefix)
		if start == -1 {
			break
		}
		start += searchFrom
		end := strings.Index(result[start+len(prefix):], suffix)
		if end == -1 {
			break
		}
		end += start + len(prefix)

		match := result[start : end+len(suffix)]
		replacement := repl(match)
		result = result[:start] + replacement + result[end+len(suffix):]
		searchFrom = start + len(replacement)
	}
	return result
}

const maxFileSize int64 = 128 * 1024 // 128 KB limit for file inclusion

func readFile(filename string) string {
	info, err := os.Stat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 {
		return ""
	}

	ext := strings.ToLower(filepath.Ext(filename))

	if reader, e := codec.GetCodec(ext); e == nil {
		text, err := reader.ReadFile(filename)
		if err != nil {
			return ""
		}
		return text // No size limit for codec-decoded files
	}

	if info.Size() > maxFileSize { // Size limit only for non-codec files
		return ""
	}

	f, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer f.Close()

	lr := io.LimitReader(f, maxFileSize+1)
	data, err := io.ReadAll(lr)
	if err != nil || int64(len(data)) > maxFileSize {
		return ""
	}

	return string(data)
}
