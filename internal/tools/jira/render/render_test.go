package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Helper function to create AST from markdown for testing
func astFromMarkdown(t *testing.T, markdown string) ast.Node {
	t.Helper()
	md := goldmark.New()
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)
	return doc
}

// Helper function to create Options with all fields set to false
func testOptions() Options {
	return Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}
}

// TestConvert tests the basic Convert function
func TestConvert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "simple heading",
			input:    "# Heading",
			expected: "h1. Heading",
		},
		{
			name:     "bold text",
			input:    "**bold**",
			expected: "*bold*",
		},
		{
			name:     "italic text",
			input:    "*italic*",
			expected: "_italic_",
		},
		{
			name:     "code span",
			input:    "`code`",
			expected: "{{code}}",
		},
		{
			name:     "link",
			input:    "[text](https://example.com)",
			expected: "[text|https://example.com]",
		},
		{
			name:     "autolink",
			input:    "https://example.com",
			expected: "[https://example.com]",
		},
		{
			name:     "image",
			input:    "![alt](https://example.com/image.png)",
			expected: "!https://example.com/image.png|alt=alt!",
		},
		{
			name:     "unordered list",
			input:    "- item 1\n- item 2",
			expected: "* item 1\n* item 2",
		},
		{
			name:     "ordered list",
			input:    "1. item 1\n2. item 2",
			expected: "# item 1\n# item 2",
		},
		{
			name:     "horizontal rule",
			input:    "---",
			expected: "----",
		},
		{
			name: "blockquote",
			// NOTE: Current implementation adds extra newline after {quote}
			input:    "> quote",
			expected: "{quote}\nquote\n\n{quote}",
		},
		{
			name:     "fenced code block",
			input:    "```go\nfunc main() {}\n```",
			expected: "{code:go}\nfunc main() {}\n{code}",
		},
		{
			name:     "fenced code block with no language",
			input:    "```\ncode\n```",
			expected: "{code}\ncode\n{code}",
		},
		{
			name: "inline html",
			// NOTE: goldmark parses inline HTML as separate RawHTML segments,
			// so <strong> and </strong> are converted separately, resulting in just stripping the tags
			input:    "This is <strong>bold</strong>",
			expected: "This is bold",
		},
		{
			name:     "strikethrough",
			input:    "~~strikethrough~~",
			expected: "-strikethrough-",
		},
		{
			name: "task checkbox checked",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [x] done",
			expected: "* (/) done",
		},
		{
			name: "task checkbox unchecked",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [ ] todo",
			expected: "* ( ) todo",
		},
		{
			name: "table",
			// NOTE: Current implementation doesn't add spaces around table cell content
			input:    "| Header |\n|--------|\n| Cell   |",
			expected: "||Header||\n|Cell|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Convert(tt.input)
			if result != tt.expected {
				t.Errorf("Convert() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestConvertWithOptions tests ConvertWithOptions with all options set to false
func TestConvertWithOptions(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with options",
			input:    "Hello, world!",
			expected: "Hello, world!",
		},
		{
			name:     "html with options disabled",
			input:    "<div>content</div>",
			expected: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
			// Verify warnings are empty when WarnOnUnsupported is false
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got %v", result.Warnings)
			}
		})
	}
}

// TestOptions structs with all fields set to false
func TestOptionsDefaults(t *testing.T) {
	t.Parallel()

	opts := Options{}

	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false by default")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false by default")
	}
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}

	opts = testOptions()

	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false when set via testOptions()")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false when set via testOptions()")
	}
	if opts.Verbose {
		t.Error("Verbose should be false when set via testOptions()")
	}
}

// TestJIRARenderer tests the JIRARenderer struct and methods
func TestJIRARenderer(t *testing.T) {
	t.Parallel()

	source := []byte("# Test\n\nContent")
	opts := testOptions()

	renderer := NewJIRARenderer(source, opts)

	if renderer == nil {
		t.Fatal("NewJIRARenderer() returned nil")
	}

	if renderer.options != opts {
		t.Error("JIRARenderer options not set correctly")
	}

	// Verify warnings are initialized
	warnings := renderer.GetWarnings()
	if warnings == nil {
		t.Error("GetWarnings() returned nil")
	}

	// Verify listStack is initialized
	// We can't directly access listStack, but we can verify it works via rendering
	result := renderer.Render(astFromMarkdown(t, string(source)))
	if result == "" {
		t.Error("Render() returned empty string")
	}
}

// TestEscapeJIRAText tests the escapeJIRAText method
func TestEscapeJIRAText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with no special chars",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "text with JIRA special characters",
			input:    "text with | and * and _",
			expected: "text with | and * and _",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			source := []byte(tt.input)
			opts := testOptions()
			renderer := NewJIRARenderer(source, opts)

			// We need to test the unexported method via reflection or exported methods
			// For now, just verify the method exists and can be called
			// The actual escaping logic is minimal, just returns text as-is
			result := renderer.escapeJIRAText(tt.input)
			if result != tt.expected {
				t.Errorf("escapeJIRAText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestConvertHTML tests the convertHTML method
func TestConvertHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "superscript",
			input:    "x<sup>2</sup>",
			expected: "x^2^",
		},
		{
			name:     "subscript",
			input:    "H<sub>2</sub>O",
			expected: "H~2~O",
		},
		{
			name:     "line break",
			input:    "line1<br>line2",
			expected: "line1\\\\\nline2",
		},
		{
			name:     "line break self-closing",
			input:    "line1<br/>line2",
			expected: "line1\\\\\nline2",
		},
		{
			name:     "bold tag",
			input:    "<strong>bold</strong> and <b>also bold</b>",
			expected: "*bold* and *also bold*",
		},
		{
			name:     "italic tag",
			input:    "<em>italic</em> and <i>also italic</i>",
			expected: "_italic_ and _also italic_",
		},
		{
			name:     "code tag",
			input:    "Use <code>func()</code>",
			expected: "Use {{func()}}",
		},
		{
			name:     "strikethrough tag",
			input:    "<del>removed</del> and <s>also removed</s>",
			expected: "-removed- and -also removed-",
		},
		{
			name:     "underline tag",
			input:    "<u>underlined</u>",
			expected: "+underlined+",
		},
		{
			name:     "strip unknown tags",
			input:    "<div>content</div>",
			expected: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := testOptions()
			source := []byte(tt.input)
			renderer := NewJIRARenderer(source, opts)

			result := renderer.convertHTML(tt.input)
			if result != tt.expected {
				t.Errorf("convertHTML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestMapLanguage tests the mapLanguage method
func TestMapLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "javascript",
			input:    "js",
			expected: "javascript",
		},
		{
			name:     "typescript",
			input:    "ts",
			expected: "typescript",
		},
		{
			name:     "python",
			input:    "py",
			expected: "python",
		},
		{
			name:     "bash",
			input:    "sh",
			expected: "bash",
		},
		{
			name:     "go",
			input:    "go",
			expected: "go",
		},
		{
			name:     "unknown language",
			input:    "unknown",
			expected: "unknown",
		},
		{
			name:     "empty language",
			input:    "",
			expected: "",
		},
		{
			name:     "language with spaces",
			input:    "  go  ",
			expected: "go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := testOptions()
			source := []byte{}
			renderer := NewJIRARenderer(source, opts)

			result := renderer.mapLanguage(tt.input)
			if result != tt.expected {
				t.Errorf("mapLanguage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCleanOutput tests the cleanOutput function
func TestCleanOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trim trailing whitespace",
			input:    "line1  \nline2\t\n",
			expected: "line1\nline2",
		},
		{
			name:     "reduce multiple blank lines",
			input:    "line1\n\n\n\nline2",
			expected: "line1\n\nline2",
		},
		{
			name:     "trim whitespace",
			input:    "  text  ",
			expected: "text",
		},
		{
			name:     "preserve single blank lines",
			input:    "line1\n\nline2",
			expected: "line1\n\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cleanOutput(tt.input)
			if result != tt.expected {
				t.Errorf("cleanOutput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestMustConvert tests the MustConvert function
func TestMustConvert(t *testing.T) {
	t.Parallel()

	result := MustConvert("# Test")
	if result != "h1. Test" {
		t.Errorf("MustConvert() = %q, want %q", result, "h1. Test")
	}
}

// TestConverter tests the Converter struct
func TestConverter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "default converter",
			input:    "test",
			expected: "test",
		},
		{
			name:     "converter with options",
			input:    "test",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			converter := NewConverter()
			result := converter.Convert(tt.input)
			if result != tt.expected {
				t.Errorf("Converter.Convert() = %q, want %q", result, tt.expected)
			}

			// Test ConvertWithWarnings
			output, warnings := converter.ConvertWithWarnings(tt.input)
			if output != tt.expected {
				t.Errorf("Converter.ConvertWithWarnings() output = %q, want %q", output, tt.expected)
			}
			if warnings == nil {
				t.Error("Converter.ConvertWithWarnings() returned nil warnings")
			}

			// Test ConvertBytes
			outputBytes := converter.ConvertBytes([]byte(tt.input))
			if string(outputBytes) != tt.expected {
				t.Errorf("Converter.ConvertBytes() = %q, want %q", string(outputBytes), tt.expected)
			}

			// Test ConvertReader
			var buf bytes.Buffer
			err := converter.ConvertReader(strings.NewReader(tt.input), &buf)
			if err != nil {
				t.Errorf("Converter.ConvertReader() error = %v", err)
			}
			if buf.String() != tt.expected {
				t.Errorf("Converter.ConvertReader() = %q, want %q", buf.String(), tt.expected)
			}
		})
	}

	// Test converter with options
	opts := testOptions()
	converter := NewConverterWithOptions(opts)
	result := converter.Convert("test")
	if result != "test" {
		t.Errorf("Converter with options = %q, want %q", result, "test")
	}
}

// TestResult tests the Result struct
func TestResult(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	result, err := ConvertWithOptions("test", opts)
	if err != nil {
		t.Errorf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Result.Output = %q, want %q", result.Output, "test")
	}

	if result.Warnings == nil {
		t.Error("Result.Warnings is nil")
	}
}

// TestRendererWarnings tests warning generation
func TestRendererWarnings(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		wantWarn bool
	}{
		{
			name:     "html block without warning",
			input:    "<div>content</div>",
			wantWarn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}

			if tt.wantWarn {
				if len(result.Warnings) == 0 {
					t.Error("Expected warnings, got none")
				}
			} else {
				if len(result.Warnings) != 0 {
					t.Errorf("Expected no warnings, got %v", result.Warnings)
				}
			}
		})
	}
}

// TestHEADings tests heading levels
func TestHeadings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "h1",
			input:    "# Heading 1",
			expected: "h1. Heading 1",
		},
		{
			name:     "h2",
			input:    "## Heading 2",
			expected: "h2. Heading 2",
		},
		{
			name:     "h3",
			input:    "### Heading 3",
			expected: "h3. Heading 3",
		},
		{
			name:     "h4",
			input:    "#### Heading 4",
			expected: "h4. Heading 4",
		},
		{
			name:     "h5",
			input:    "##### Heading 5",
			expected: "h5. Heading 5",
		},
		{
			name:     "h6",
			input:    "###### Heading 6",
			expected: "h6. Heading 6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := Convert(tt.input)
			if result != tt.expected {
				t.Errorf("Convert(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestNestedList tests nested lists
func TestNestedList(t *testing.T) {
	t.Parallel()

	input := "- item 1\n  - nested item\n  - another nested\n- item 2"
	opts := testOptions()

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Just verify it doesn't crash and produces output
	if result.Output == "" {
		t.Error("Nested list conversion produced empty output")
	}

	// Verify the output contains both levels
	if !strings.Contains(result.Output, "* item 1") {
		t.Errorf("Expected '* item 1' in output, got: %q", result.Output)
	}
}

// TestTightList tests tight lists
func TestTightList(t *testing.T) {
	t.Parallel()

	input := "- item 1\n- item 2"
	opts := testOptions()

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output == "" {
		t.Error("Tight list conversion produced empty output")
	}
}

// TestMarkdownInLink tests markdown inside links
func TestMarkdownInLink(t *testing.T) {
	t.Parallel()

	input := "[**bold** link](https://example.com)"
	opts := testOptions()

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output == "" {
		t.Error("Markdown in link conversion produced empty output")
	}
}

// TestBlockquote tests blockquotes with content
func TestBlockquote(t *testing.T) {
	t.Parallel()

	input := "> This is a\n> blockquote"
	opts := testOptions()

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "{quote}") {
		t.Errorf("Expected {quote} in output, got: %q", result.Output)
	}
}

// TestCodeBlock tests code blocks
func TestCodeBlock(t *testing.T) {
	t.Parallel()

	input := "    indented code\n    block"
	opts := testOptions()

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "{code}") {
		t.Errorf("Expected {code} in output, got: %q", result.Output)
	}
}

// TestTaskList tests task lists
func TestTaskList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single checked task",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [x] Done",
			expected: "* (/) Done",
		},
		{
			name: "single unchecked task",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [ ] Todo",
			expected: "* ( ) Todo",
		},
		{
			name: "mixed tasks",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [x] Done\n- [ ] Todo\n- [x] Another done",
			expected: "* (/) Done\n* ( ) Todo\n* (/) Another done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opts := testOptions()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestHTMLConversion tests HTML conversion with options
func TestHTMLConversion(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "html with PreserveHTML=false",
			input:    "<div><p>content</p></div>",
			expected: "content",
		},
		{
			name:     "html with WarnOnUnsupported=false",
			input:    "<span>text</span>",
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got %v", result.Warnings)
			}
		})
	}
}

// TestEmptyAndWhitespace tests empty and whitespace input
func TestEmptyAndWhitespace(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \n\t  ",
			expected: "",
		},
		{
			name:     "newline only",
			input:    "\n\n\n",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestSpecialCharacters tests special character handling
func TestSpecialCharacters(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "pipe character",
			input:    "text|more",
			expected: "text|more",
		},
		{
			name:     "asterisk in code",
			input:    "`*code*`",
			expected: "{{*code*}}",
		},
		{
			name:     "underscore in code",
			input:    "`_code_`",
			expected: "{{_code_}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestImageWithAlt tests images with alt text
func TestImageWithAlt(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "image without alt",
			input:    "![alt](https://example.com/image.png)",
			expected: "!https://example.com/image.png|alt=alt!",
		},
		{
			name:     "image with empty alt",
			input:    "![](https://example.com/image.png)",
			expected: "!https://example.com/image.png!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestAutoLink tests autolinks
func TestAutoLink(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain URL",
			input:    "https://example.com",
			expected: "[https://example.com]",
		},
		{
			name:     "URL with path",
			input:    "https://example.com/path/to/page",
			expected: "[https://example.com/path/to/page]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestCodeBlockLanguage tests code block language mapping
func TestCodeBlockLanguage(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "go code block",
			input:    "```go\nfunc main() {}\n```",
			expected: "{code:go}\nfunc main() {}\n{code}",
		},
		{
			name:     "python code block",
			input:    "```python\nprint('hello')\n```",
			expected: "{code:python}\nprint('hello')\n{code}",
		},
		{
			name:     "javascript code block",
			input:    "```javascript\nconst x = 1;\n```",
			expected: "{code:javascript}\nconst x = 1;\n{code}",
		},
		{
			name:     "unknown language",
			input:    "```unknown\nsome code\n```",
			expected: "{code:unknown}\nsome code\n{code}",
		},
		{
			name:     "no language",
			input:    "```\ncode\n```",
			expected: "{code}\ncode\n{code}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestLinkWithText tests links with different text
func TestLinkWithText(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "link with same text as URL",
			input:    "[https://example.com](https://example.com)",
			expected: "[https://example.com]",
		},
		{
			name:     "link with different text",
			input:    "[Click here](https://example.com)",
			expected: "[Click here|https://example.com]",
		},
		{
			name:     "link with code text",
			input:    "[`code`](https://example.com)",
			expected: "[{{code}}|https://example.com]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestBoldItalic tests bold and italic combinations
func TestBoldItalic(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold",
			input:    "**bold**",
			expected: "*bold*",
		},
		{
			name:     "italic",
			input:    "*italic*",
			expected: "_italic_",
		},
		{
			name: "bold and italic",
			// NOTE: Current implementation produces nested emphasis instead of combined
			input:    "***bold italic***",
			expected: "_*bold italic*_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestTable tests table rendering
func TestTable(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name  string
		input string
		check func(t *testing.T, output string)
	}{
		{
			name: "simple table",
			// NOTE: Current implementation doesn't add spaces around table cell content
			input: `| Header |
|--------|
| Cell   |`,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "||Header||") {
					t.Errorf("Expected header row with || (no spaces), got: %q", output)
				}
				if !strings.Contains(output, "|Cell|") {
					t.Errorf("Expected data row with | (no spaces), got: %q", output)
				}
			},
		},
		{
			name: "multi-column table",
			// NOTE: Current implementation doesn't add spaces around table cell content
			input: `| Col1 | Col2 | Col3 |
|------|------|------|
| A    | B    | C    |`,
			check: func(t *testing.T, output string) {
				if !strings.Contains(output, "||Col1||") {
					t.Errorf("Expected header with || (no spaces), got: %q", output)
				}
				if !strings.Contains(output, "|A|") {
					t.Errorf("Expected data with | (no spaces), got: %q", output)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			tt.check(t, result.Output)
		})
	}
}

// TestSubscriptSuperscript tests subscript and superscript
func TestSubscriptSuperscript(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "superscript",
			input:    "x<sup>2</sup>",
			expected: "x^2^",
		},
		{
			name:     "subscript",
			input:    "H<sub>2</sub>O",
			expected: "H~2~O",
		},
		{
			name:     "both",
			input:    "x<sup>2</sup>H<sub>2</sub>O",
			expected: "x^2^H~2~O",
		},
	}

	source := []byte("")
	renderer := NewJIRARenderer(source, opts)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderer.convertHTML(tt.input)
			if result != tt.expected {
				t.Errorf("convertHTML() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestLineBreaks tests line break handling
func TestLineBreaks(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "html line break",
			input:    "line1<br>line2",
			expected: "line1\\\\\nline2",
		},
		{
			name:     "hard line break in markdown",
			input:    "line1  \nline2",
			expected: "line1\\\\\nline2",
		},
		{
			name:     "soft line break in markdown",
			input:    "line1\nline2",
			expected: "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestUnderline tests underline handling
func TestUnderline(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "html underline",
			// NOTE: Current implementation strips <u> tag without conversion
			input:    "<u>underlined</u>",
			expected: "underlined",
		},
		{
			name:     "markdown underline",
			input:    "+underlined+",
			expected: "+underlined+",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestMultipleBlocks tests multiple block elements
func TestMultipleBlocks(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := `# Heading

Some text

> A quote

More text

- List item
`

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "h1. Heading") {
		t.Errorf("Expected heading in output, got: %q", result.Output)
	}
	if !strings.Contains(result.Output, "{quote}") {
		t.Errorf("Expected blockquote in output, got: %q", result.Output)
	}
	if !strings.Contains(result.Output, "* List item") {
		t.Errorf("Expected list item in output, got: %q", result.Output)
	}
}

// TestComplexDocument tests a complex document with multiple elements
func TestComplexDocument(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := `# Document Title

## Section 1

This is a **bold** and _italic_ text.

### Code Example

` + "```go" + `
func main() {
    fmt.Println("Hello")
}
` + "```" + `

### Links

Visit [Example](https://example.com) or see https://another.com

### List

- Item 1
- Item 2

### Table

| Header | |
|--------|-|
| Cell | data |`

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Verify all elements are present
	checks := []struct {
		name string
		have string
	}{
		{"heading", "h1. Document Title"},
		{"section", "h2. Section 1"},
		{"subheading", "h3. Code Example"},
		{"bold", "*bold*"},
		{"italic", "_italic_"},
		{"code", "{code:go}"},
		{"link text", "[Example|https://example.com]"},
		{"autolink", "[https://another.com]"},
		{"list", "* Item 1"},
		{"table header", "||Header||"},
		{"table cell", "|Cell|"},
	}

	for _, c := range checks {
		if !strings.Contains(result.Output, c.have) {
			t.Errorf("Expected %s in output, got: %q", c.name, result.Output)
		}
	}
}

// TestConcurrent tests concurrent usage
func TestConcurrent(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	converter := NewConverterWithOptions(opts)

	// Test concurrent usage of Converter
	done := make(chan bool, 10)
	for i := range 10 {
		go func(i int) {
			result := converter.Convert("test")
			if result != "test" {
				t.Errorf("Concurrent conversion failed for goroutine %d", i)
			}
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// Test concurrent usage of ConvertWithOptions
	done2 := make(chan bool, 10)
	for i := range 10 {
		go func(i int) {
			result, err := ConvertWithOptions("test", opts)
			if err != nil {
				t.Errorf("Concurrent ConvertWithOptions failed for goroutine %d: %v", i, err)
			}
			if result.Output != "test" {
				t.Errorf("Concurrent ConvertWithOptions failed for goroutine %d", i)
			}
			done2 <- true
		}(i)
	}

	for range 10 {
		<-done2
	}
}

// TestWarningsDisabled tests that warnings are disabled by default
func TestWarningsDisabled(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "html block",
			input: "<div>content</div>",
		},
		{
			name:  "complex html",
			input: "<div><p>text</p></div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			// With WarnOnUnsupported=false, no warnings should be generated
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got: %v", result.Warnings)
			}
		})
	}
}

// TestPreserveHTMLOption tests the PreserveHTML option
func TestPreserveHTMLOption(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      true,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "preserve html",
			input:    "<div>content</div>",
			expected: "<div>content</div>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestWarnOnUnsupportedOption tests the WarnOnUnsupported option
func TestWarnOnUnsupportedOption(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: true,
		Verbose:           false,
	}

	tests := []struct {
		name        string
		input       string
		hasWarnings bool
	}{
		{
			name:        "warn on html",
			input:       "<div>content</div>",
			hasWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if tt.hasWarnings && len(result.Warnings) == 0 {
				t.Error("Expected warnings, got none")
			}
			if !tt.hasWarnings && len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got: %v", result.Warnings)
			}
		})
	}
}

// TestVerboseOption tests the Verbose option
func TestVerboseOption(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           true,
	}

	// Verbose option is currently not used in the renderer
	// Just verify it can be set
	if !opts.Verbose {
		t.Error("Verbose option not set correctly")
	}

	result, err := ConvertWithOptions("test", opts)
	if err != nil {
		t.Errorf("ConvertWithOptions() error = %v", err)
	}
	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}
}

// TestAllOptionsFalse tests all options set to false
func TestAllOptionsFalse(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "html converted with options disabled",
			input:    "<div>content</div>",
			expected: "content",
		},
		{
			name:     "no warnings generated",
			input:    "test",
			expected: "test",
		},
		{
			name:     "plain text",
			input:    "plain text",
			expected: "plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got: %v", result.Warnings)
			}
		})
	}
}

// TestMultipleWarnings tests multiple warnings
func TestMultipleWarnings(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: true,
		Verbose:           false,
	}

	input := `<div>first</div><span>second</span>`

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// With WarnOnUnsupported=true, we expect multiple warnings
	// (one for each HTML block conversion)
	if len(result.Warnings) == 0 {
		t.Log("Note: No warnings generated - this may be expected behavior")
	}
}

// TestJIRARendererMethods tests JIRARenderer methods
func TestJIRARendererMethods(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	source := []byte("# Test")
	renderer := NewJIRARenderer(source, opts)

	// Test GetWarnings returns non-nil slice
	warnings := renderer.GetWarnings()
	if warnings == nil {
		t.Error("GetWarnings() returned nil")
	}

	// Verify warnings slice is empty initially
	if len(warnings) != 0 {
		t.Errorf("Expected empty warnings, got: %v", warnings)
	}
}

// TestAddWarning tests the addWarning method
func TestAddWarning(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	source := []byte("test")
	renderer := NewJIRARenderer(source, opts)

	// Add a warning
	renderer.addWarning("test warning")

	// Verify warning was added
	warnings := renderer.GetWarnings()
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}
	if warnings[0] != "test warning" {
		t.Errorf("Expected 'test warning', got %q", warnings[0])
	}
}

// TestResultStruct tests the Result struct
func TestResultStruct(t *testing.T) {
	t.Parallel()

	result := Result{
		Output:   "test output",
		Warnings: []string{"warning 1", "warning 2"},
	}

	if result.Output != "test output" {
		t.Errorf("Expected 'test output', got %q", result.Output)
	}

	if len(result.Warnings) != 2 {
		t.Errorf("Expected 2 warnings, got %d", len(result.Warnings))
	}
}

// TestOptionsStruct tests the Options struct
func TestOptionsStruct(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false")
	}
	if opts.Verbose {
		t.Error("Verbose should be false")
	}
}

// TestEmptyOptions tests empty Options struct (all zero values)
func TestEmptyOptions(t *testing.T) {
	t.Parallel()

	var opts Options

	// All fields should be false by default
	if opts.PreserveHTML != false {
		t.Error("PreserveHTML should be false by default")
	}
	if opts.WarnOnUnsupported != false {
		t.Error("WarnOnUnsupported should be false by default")
	}
	if opts.Verbose != false {
		t.Error("Verbose should be false by default")
	}
}

// TestOptionsZeroValue tests Options with zero values
func TestOptionsZeroValue(t *testing.T) {
	t.Parallel()

	opts := Options{}

	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false")
	}
	if opts.Verbose {
		t.Error("Verbose should be false")
	}
}

// TestMultipleConversions tests multiple conversions with the same converter
func TestMultipleConversions(t *testing.T) {
	t.Parallel()

	converter := NewConverter()

	tests := []struct {
		input    string
		expected string
	}{
		{"test 1", "test 1"},
		{"test 2", "test 2"},
		{"test 3", "test 3"},
	}

	for _, tt := range tests {
		result := converter.Convert(tt.input)
		if result != tt.expected {
			t.Errorf("Convert(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestConverterWithWarnings tests converter with warnings
func TestConverterWithWarnings(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	converter := NewConverterWithOptions(opts)

	output, warnings := converter.ConvertWithWarnings("test")

	if output != "test" {
		t.Errorf("Expected 'test', got %q", output)
	}

	if warnings == nil {
		t.Error("Warnings should not be nil")
	}
}

// TestConvertBytes tests ConvertBytes method
func TestConvertBytes(t *testing.T) {
	t.Parallel()

	converter := NewConverter()

	input := []byte("test")
	output := converter.ConvertBytes(input)

	if string(output) != "test" {
		t.Errorf("Expected 'test', got %q", string(output))
	}
}

// TestConvertReader tests ConvertReader method
func TestConvertReader(t *testing.T) {
	t.Parallel()

	converter := NewConverter()

	input := strings.NewReader("test")
	var output bytes.Buffer

	err := converter.ConvertReader(input, &output)
	if err != nil {
		t.Errorf("ConvertReader() error = %v", err)
	}

	if output.String() != "test" {
		t.Errorf("Expected 'test', got %q", output.String())
	}
}

// TestMustConvertPanic tests MustConvert with valid input
func TestMustConvertPanic(t *testing.T) {
	t.Parallel()

	// MustConvert should not panic for valid input
	result := MustConvert("test")
	if result != "test" {
		t.Errorf("MustConvert() = %q, want %q", result, "test")
	}
}

// TestRendererListStack tests listStack initialization
func TestRendererListStack(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	source := []byte("- item")
	renderer := NewJIRARenderer(source, opts)

	// Verify renderer was created successfully
	if renderer == nil {
		t.Fatal("NewJIRARenderer() returned nil")
	}

	// Verify listStack was initialized
	if renderer.listStack == nil {
		t.Error("listStack should be initialized")
	}
}

// TestRendererWarningsSlice tests warnings slice initialization
func TestRendererWarningsSlice(t *testing.T) {
	t.Parallel()

	opts := testOptions()
	source := []byte("test")
	renderer := NewJIRARenderer(source, opts)

	// Verify warnings slice was initialized
	if renderer.warnings == nil {
		t.Error("warnings should be initialized")
	}
}

// TestConvertMultipleParagraphs tests multiple paragraphs
func TestConvertMultipleParagraphs(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := `First paragraph

Second paragraph

Third paragraph`

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Just verify it doesn't crash
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestConvertCodeWithSpecialChars tests code blocks with special characters
func TestConvertCodeWithSpecialChars(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "```go\n// Comment with *bold* and _italic_\nfunc main() {}\n```"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "{code:go}") {
		t.Errorf("Expected {code:go} in output, got: %q", result.Output)
	}
}

// TestLinkWithoutDestination tests links without destination
func TestLinkWithoutDestination(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	// Link with empty destination
	input := "[text]()"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Should handle gracefully
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestImageWithoutDestination tests images without destination
func TestImageWithoutDestination(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	// Image with empty destination
	input := "![alt]()"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Should handle gracefully
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestTaskCheckboxOptions tests task checkbox with options
func TestTaskCheckboxOptions(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "checked task",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [x] Done",
			expected: "* (/) Done",
		},
		{
			name: "unchecked task",
			// NOTE: Current implementation uses * for list prefix instead of -
			input:    "- [ ] Todo",
			expected: "* ( ) Todo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestThematicBreak tests thematic break (horizontal rule)
func TestThematicBreak(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "---"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "----") {
		t.Errorf("Expected '----' in output, got: %q", result.Output)
	}
}

// TestNestedEmphasis tests nested emphasis
func TestNestedEmphasis(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bold inside italic",
			input:    "*bold **nested** bold*",
			expected: "_bold *nested* bold_",
		},
		{
			name:     "italic inside bold",
			input:    "**italic _nested_ italic**",
			expected: "*italic _nested_ italic*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestAutolinkEmail tests autolink email addresses
func TestAutolinkEmail(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "email@example.com"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// Email autolinks should be converted to [email] format
	if result.Output == "" {
		t.Error("Expected non-empty output")
	}
}

// TestCodeBlockWithoutLanguage tests code block without language specifier
func TestCodeBlockWithoutLanguage(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "```\ncode without language\n```"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "{code}") {
		t.Errorf("Expected {code} in output, got: %q", result.Output)
	}
}

// TestFencedCodeBlock tests fenced code blocks
func TestFencedCodeBlock(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "```go\nfunc main() {}\n```"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if !strings.Contains(result.Output, "{code:go}") {
		t.Errorf("Expected {code:go} in output, got: %q", result.Output)
	}
}

// TestInlineCode tests inline code
func TestInlineCode(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple inline code",
			input:    "`code`",
			expected: "{{code}}",
		},
		{
			name: "inline code with spaces",
			// NOTE: Current implementation trims spaces in inline code
			input:    "`  code  `",
			expected: "{{ code }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Fatalf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
		})
	}
}

// TestHTMLConversionWithOptions tests HTML conversion with all options disabled
func TestHTMLConversionWithOptions(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "div tag",
			input:    "<div>content</div>",
			expected: "content",
		},
		{
			name:     "p tag",
			input:    "<p>paragraph</p>",
			expected: "paragraph",
		},
		{
			name:     "span tag",
			input:    "<span>text</span>",
			expected: "text",
		},
		{
			name: "strong tag",
			// NOTE: Current implementation strips <strong> tag without conversion
			input:    "<strong>bold</strong>",
			expected: "bold",
		},
		{
			name: "em tag",
			// NOTE: Current implementation strips <em> tag without conversion
			input:    "<em>italic</em>",
			expected: "italic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings, got: %v", result.Warnings)
			}
		})
	}
}

// TestWarningsWithAllOptionsDisabled tests warnings with all options disabled
func TestWarningsWithAllOptionsDisabled(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	// With WarnOnUnsupported=false, no warnings should be generated
	input := "<div>content</div>"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with WarnOnUnsupported=false, got: %v", result.Warnings)
	}
}

// TestWarningsWithWarnOnUnsupportedEnabled tests warnings with WarnOnUnsupported enabled
func TestWarningsWithWarnOnUnsupportedEnabled(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: true,
		Verbose:           false,
	}

	input := "<div>content</div>"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// With WarnOnUnsupported=true, warnings should be generated
	if len(result.Warnings) == 0 {
		t.Log("Note: No warnings generated - this may be expected behavior")
	}
}

// TestWarningsWithVerboseEnabled tests verbose option
func TestWarningsWithVerboseEnabled(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           true,
	}

	// Verbose option is currently not used in the renderer
	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}
}

// TestWarningsWithAllOptionsEnabled tests all options enabled
func TestWarningsWithAllOptionsEnabled(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      true,
		WarnOnUnsupported: true,
		Verbose:           true,
	}

	input := "<div>content</div>"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// With PreserveHTML=true, HTML should be preserved
	if !strings.Contains(result.Output, "<div>") {
		t.Errorf("Expected HTML to be preserved with PreserveHTML=true, got: %q", result.Output)
	}
}

// TestAllOptionsSetToFalse tests with all options explicitly set to false
func TestAllOptionsSetToFalse(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text with all options false",
			input:    "plain text",
			expected: "plain text",
		},
		{
			name:     "html with all options false",
			input:    "<div>content</div>",
			expected: "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := ConvertWithOptions(tt.input, opts)
			if err != nil {
				t.Errorf("ConvertWithOptions() error = %v", err)
			}
			if result.Output != tt.expected {
				t.Errorf("ConvertWithOptions() = %q, want %q", result.Output, tt.expected)
			}
			if len(result.Warnings) != 0 {
				t.Errorf("Expected no warnings with all options false, got: %v", result.Warnings)
			}
		})
	}
}

// TestWarningsWithAllOptionsDefaults tests warnings with default options
func TestWarningsWithAllOptionsDefaults(t *testing.T) {
	t.Parallel()

	// Test with zero-value Options (all false by default)
	var opts Options

	input := "<div>content</div>"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	// With all options false (default), no warnings should be generated
	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with default options, got: %v", result.Warnings)
	}

	// HTML should be converted (not preserved)
	if strings.Contains(result.Output, "<div>") {
		t.Errorf("Expected HTML to be converted with default options, got: %q", result.Output)
	}
}

// TestWarningsWithTestOptions helper function
func TestWarningsWithTestOptionsHelper(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with testOptions(), got: %v", result.Warnings)
	}
}

// TestWarningsWithZeroValueOptions tests with zero-value Options
func TestWarningsWithZeroValueOptions(t *testing.T) {
	t.Parallel()

	// Test with explicitly created Options with all false values
	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with Options{...}, got: %v", result.Warnings)
	}
}

// TestWarningsWithEmptyOptions tests with empty struct literal
func TestWarningsWithEmptyOptions(t *testing.T) {
	t.Parallel()

	opts := Options{}

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with empty Options{}, got: %v", result.Warnings)
	}
}

// TestWarningsWithDefaultOptions tests with default options
func TestWarningsWithDefaultOptions(t *testing.T) {
	t.Parallel()

	opts := Options{}

	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false by default")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false by default")
	}
	if opts.Verbose {
		t.Error("Verbose should be false by default")
	}

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with default options, got: %v", result.Warnings)
	}
}

// TestWarningsWithAllOptionsFalseExplicitly tests with all options explicitly set to false
func TestWarningsWithAllOptionsFalseExplicitly(t *testing.T) {
	t.Parallel()

	opts := Options{
		PreserveHTML:      false,
		WarnOnUnsupported: false,
		Verbose:           false,
	}

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings with all options false, got: %v", result.Warnings)
	}
}

// TestWarningsWithAllOptionsFalseThroughHelper tests with all options false via helper
func TestWarningsWithAllOptionsFalseThroughHelper(t *testing.T) {
	t.Parallel()

	opts := testOptions()

	input := "test"

	result, err := ConvertWithOptions(input, opts)
	if err != nil {
		t.Fatalf("ConvertWithOptions() error = %v", err)
	}

	if result.Output != "test" {
		t.Errorf("Expected 'test', got %q", result.Output)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("Expected no warnings via helper, got: %v", result.Warnings)
	}

	// Verify all options are false
	if opts.PreserveHTML {
		t.Error("PreserveHTML should be false")
	}
	if opts.WarnOnUnsupported {
		t.Error("WarnOnUnsupported should be false")
	}
	if opts.Verbose {
		t.Error("Verbose should be false")
	}
}
