package printer

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// defaultMuTest serializes the tests that swap the package default printer. They can still run
// in parallel with the other tests, but two of them at once would print into each other's buffer.
var defaultMuTest sync.Mutex

// swapDefault takes defaultMuTest and returns a function that restores the original default printer
// and releases it.
func swapDefault(t *testing.T) func() {
	t.Helper()
	defaultMuTest.Lock()
	original := GetDefault()
	return func() {
		SetDefault(original)
		defaultMuTest.Unlock()
	}
}

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, false)

	if p == nil {
		t.Fatal("New() returned nil")
	}

	// Verify it's the correct type
	_, ok := p.(*printer)
	if !ok {
		t.Fatalf("New() returned wrong type: %T", p)
	}
}

func TestVerbose(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, false)

	// Test verbose off - Info should output (only Debug and ToolCall respect verbose)
	p.Info("info message")
	output := buf.String()
	if output == "" {
		t.Error("Expected output from Info even when verbose is off")
	}

	// Test verbose on
	p.Verbose(true)
	buf.Reset()

	p.Info("info message")
	output = buf.String()
	if output == "" {
		t.Error("Expected output when verbose is on")
	}

	// Test Debug with verbose off
	buf.Reset()
	p.Verbose(false)
	p.Debug("debug message")
	output = buf.String()
	if output != "" {
		t.Errorf("Expected no output from Debug when verbose is off, got: %q", output)
	}

	// Test Debug with verbose on
	buf.Reset()
	p.Verbose(true)
	p.Debug("debug message")
	output = buf.String()
	if output == "" {
		t.Error("Expected output from Debug when verbose is on")
	}
}

func TestDebug(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		message     string
		params      []any
		expectEmpty bool
	}{
		{
			name:        "debug with verbose off",
			verbose:     false,
			message:     "debug message",
			expectEmpty: true,
		},
		{
			name:        "debug with verbose on",
			verbose:     true,
			message:     "debug message",
			expectEmpty: false,
		},
		{
			name:        "debug with params",
			verbose:     true,
			message:     "value=%d",
			params:      []any{42},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := New(&buf, tt.verbose)

			p.Debug(tt.message, tt.params...)

			output := buf.String()
			if tt.expectEmpty && output != "" {
				t.Errorf("Expected no output, got: %q", output)
			}
			if !tt.expectEmpty && output == "" {
				t.Error("Expected output")
			}

			// Check message content
			if !tt.expectEmpty {
				if !strings.Contains(output, tt.message) {
					t.Errorf("Output doesn't contain expected message: %q", output)
				}
			}
		})
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	p.Info("info message")
	output := buf.String()

	if output == "" {
		t.Error("Expected output")
	}

	if !strings.Contains(output, "info message") {
		t.Errorf("Output doesn't contain expected message: %q", output)
	}
}

func TestInfo_Params(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	// Note: Info just uses the message as-is without format interpretation
	// when params are provided, they go into the [key=value] section
	p.Info("value")
	output := buf.String()

	// Just check that the message appears
	if !strings.Contains(output, "value") {
		t.Errorf("Output doesn't contain expected message: %q", output)
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	p.Warn("warning message")
	output := buf.String()

	if output == "" {
		t.Error("Expected output")
	}

	if !strings.Contains(output, string(IconAlert)) {
		t.Errorf("Output doesn't contain warning icon: %q", output)
	}

	if !strings.Contains(output, "warning message") {
		t.Errorf("Output doesn't contain expected message: %q", output)
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	p.Error("error message")
	output := buf.String()

	if output == "" {
		t.Error("Expected output")
	}

	if !strings.Contains(output, string(IconError)) {
		t.Errorf("Output doesn't contain error icon: %q", output)
	}

	if !strings.Contains(output, "error message") {
		t.Errorf("Output doesn't contain expected message: %q", output)
	}
}

func TestToolCall(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		typeIcon    string
		message     string
		expectEmpty bool
	}{
		{
			name:        "tool call with verbose off",
			verbose:     false,
			typeIcon:    IconTool,
			message:     "tool call message",
			expectEmpty: true,
		},
		{
			name:        "tool call with verbose on",
			verbose:     true,
			typeIcon:    IconTool,
			message:     "tool call message",
			expectEmpty: false,
		},
		{
			name:        "tool call with custom icon",
			verbose:     true,
			typeIcon:    IconSave,
			message:     "save message",
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := New(&buf, tt.verbose)

			p.ToolCall(tt.typeIcon, tt.message)

			output := buf.String()
			if tt.expectEmpty && output != "" {
				t.Errorf("Expected no output, got: %q", output)
			}
			if !tt.expectEmpty && output == "" {
				t.Error("Expected output")
			}

			// Check icon and message
			if !tt.expectEmpty {
				if !strings.Contains(output, tt.typeIcon) {
					t.Errorf("Output doesn't contain icon: %q", output)
				}
				if !strings.Contains(output, tt.message) {
					t.Errorf("Output doesn't contain message: %q", output)
				}
			}
		})
	}
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name       string
		typeIcon   string
		message    string
		params     []any
		wantIcon   bool
		wantParams bool
	}{
		{
			name:       "with icon",
			typeIcon:   IconRobot,
			message:    "hello",
			params:     nil,
			wantIcon:   true,
			wantParams: false,
		},
		{
			name:       "without icon",
			typeIcon:   "",
			message:    "hello",
			params:     nil,
			wantIcon:   false,
			wantParams: false,
		},
		{
			name:       "with params",
			typeIcon:   "",
			message:    "key=%s",
			params:     []any{"value1", "value2"},
			wantIcon:   false,
			wantParams: true,
		},
		{
			name:       "with icon and params",
			typeIcon:   IconSave,
			message:    "data",
			params:     []any{"a", "b"},
			wantIcon:   true,
			wantParams: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := New(&buf, true)

			p.Print(tt.typeIcon, tt.message, tt.params...)

			output := buf.String()
			if output == "" {
				t.Error("Expected output")
			}

			// Check icon
			if tt.wantIcon {
				if !strings.Contains(output, tt.typeIcon) {
					t.Errorf("Output doesn't contain icon: %q", output)
				}
			} else {
				if tt.typeIcon == "" && strings.Contains(output, string(IconError)) {
					// Make sure we're not getting IconError instead
					lines := strings.Split(output, "\n")
					// First line should not start with IconError
					firstLine := lines[0]
					if strings.HasPrefix(firstLine, string(IconError)) {
						t.Errorf("Output contains unexpected IconError: %q", output)
					}
				}
			}

			// Check message
			if !strings.Contains(output, tt.message) {
				t.Errorf("Output doesn't contain message: %q", output)
			}

			// Check params
			if tt.wantParams && len(tt.params) >= 2 {
				// Check that params are in format key=value
				paramStr := ""
				for i := 0; i < len(tt.params)/2; i++ {
					expected := tt.params[i*2].(string) + "=" + tt.params[i*2+1].(string)
					if strings.Contains(output, expected) {
						paramStr = expected
						break
					}
				}
				if paramStr == "" {
					t.Errorf("Output doesn't contain params in expected format: %q", output)
				}
			}
		})
	}
}

func TestEmptyLine(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	p.EmptyLine()
	output := buf.String()

	// Should contain just a newline
	expected := "\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestToolEmptyLine(t *testing.T) {
	tests := []struct {
		name        string
		verbose     bool
		expectEmpty bool
	}{
		{
			name:        "tool empty line with verbose off",
			verbose:     false,
			expectEmpty: true,
		},
		{
			name:        "tool empty line with verbose on",
			verbose:     true,
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			p := New(&buf, tt.verbose)

			p.ToolEmptyLine()
			output := buf.String()

			if tt.expectEmpty && output != "" {
				t.Errorf("Expected no output, got: %q", output)
			}
			if !tt.expectEmpty && output == "" {
				t.Error("Expected output")
			}
			if !tt.expectEmpty && output != "\n" {
				t.Errorf("Expected single newline, got: %q", output)
			}
		})
	}
}

func TestStaticFunctions(t *testing.T) {
	// Save original default printer
	defer swapDefault(t)()

	var buf bytes.Buffer
	SetDefault(New(&buf, true))

	// Test static Info
	Info("static info")
	output := buf.String()
	if !strings.Contains(output, "static info") {
		t.Errorf("Static Info failed: %q", output)
	}

	buf.Reset()

	// Test static Error
	Error("static error")
	output = buf.String()
	if !strings.Contains(output, string(IconError)) {
		t.Errorf("Static Error failed: %q", output)
	}

	buf.Reset()

	// Test static Warn
	Warn("static warn")
	output = buf.String()
	if !strings.Contains(output, string(IconAlert)) {
		t.Errorf("Static Warn failed: %q", output)
	}

	buf.Reset()

	// Test static Print
	Print(IconRobot, "static print")
	output = buf.String()
	if !strings.Contains(output, string(IconRobot)) {
		t.Errorf("Static Print failed: %q", output)
	}

	buf.Reset()

	// Test static ToolCall
	ToolCall(IconTool, "static tool call")
	output = buf.String()
	if !strings.Contains(output, string(IconTool)) {
		t.Errorf("Static ToolCall failed: %q", output)
	}

	buf.Reset()

	// Test static EmptyLine
	EmptyLine()
	output = buf.String()
	if output != "\n" {
		t.Errorf("Static EmptyLine failed: %q", output)
	}

	buf.Reset()

	// Test static ToolEmptyLine
	ToolEmptyLine()
	output = buf.String()
	if output != "\n" {
		t.Errorf("Static ToolEmptyLine failed: %q", output)
	}
}

func TestMultipleLines(t *testing.T) {
	var buf bytes.Buffer
	p := New(&buf, true)

	p.Info("line1")
	p.Info("line2")
	p.Info("line3")

	output := buf.String()
	// Trim trailing newline and split
	output = strings.TrimSuffix(output, "\n")
	lines := strings.Split(output, "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "line1") {
		t.Errorf("Line 1 doesn't contain 'line1': %q", lines[0])
	}
	if !strings.Contains(lines[1], "line2") {
		t.Errorf("Line 2 doesn't contain 'line2': %q", lines[1])
	}
	if !strings.Contains(lines[2], "line3") {
		t.Errorf("Line 3 doesn't contain 'line3': %q", lines[2])
	}
}

// TestPrintRaw tests the PrintRaw function with various inputs
func TestPrintRaw_SimpleText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("Hello World")
	output := buf.String()

	// When verbose=true, PrintRaw should add empty lines around the content
	// and format the content with leading space
	expected := "\n Hello World\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_WithNewlines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("line1\nline2\nline3")
	output := buf.String()

	// Newlines in the input should be replaced with "\n " for proper indentation
	expected := "\n line1\n line2\n line3\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_VerboseOff(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, false)

	p.PrintRaw("Hello World")
	output := buf.String()

	// When verbose=false, PrintRaw should output nothing
	if output != "" {
		t.Errorf("Expected empty output when verbose=false, got %q", output)
	}
}

func TestPrintRaw_MultilineWithEmptyLines(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("line1\n\nline3")
	output := buf.String()

	// Empty lines in the input should also be indented
	expected := "\n line1\n \n line3\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_StaticFunction(t *testing.T) {
	t.Parallel()

	// Save original default printer
	defer swapDefault(t)()

	var buf bytes.Buffer
	SetDefault(New(&buf, true))

	PrintRaw("test static")
	output := buf.String()

	expected := "\n test static\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_StaticFunction_VerboseOff(t *testing.T) {
	t.Parallel()

	// Save original default printer
	defer swapDefault(t)()

	var buf bytes.Buffer
	SetDefault(New(&buf, false))

	PrintRaw("test static")
	output := buf.String()

	// When verbose=false, static PrintRaw should output nothing
	if output != "" {
		t.Errorf("Expected empty output when verbose=false, got %q", output)
	}
}

func TestPrintRaw_SpecialCharacters(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("Tab\there\nNewline\there")
	output := buf.String()

	// Special characters should be preserved
	expected := "\n Tab\there\n Newline\there\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_LongText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	longText := strings.Repeat("a", 1000) + "\n" + strings.Repeat("b", 1000)
	p.PrintRaw(longText)
	output := buf.String()

	// Should handle long text without issues
	if !strings.Contains(output, " aaaa") {
		t.Errorf("Output doesn't contain expected long text")
	}
}

func TestPrintRaw_EmptyString(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("")
	output := buf.String()

	// Even empty string should get the empty line treatment
	// PrintRaw with empty string: "\n" (empty line) + " " (space for empty data) + "\n" + "\n" (empty line)
	// But the data is empty, so it's: "\n" + "\n" + "\n" = "\n\n\n"
	// Actually, the code does: fmt.Fprintln(p.out, " " + "")
	// So it outputs: "\n" (first empty line) + " \n" (space + data + newline) + "\n" (second empty line)
	// = "\n \n\n"
	expected := "\n \n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrintRaw_Cyrillic(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	p.PrintRaw("Привет, мир!")
	output := buf.String()

	expected := "\n Привет, мир!\n\n"
	if output != expected {
		t.Errorf("Expected %q, got %q", expected, output)
	}
}

func TestPrinter_ConcurrentUse(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	p := New(&buf, true)

	const workers, rounds = 8, 50
	var wg sync.WaitGroup
	for i := range workers {
		wg.Go(func() {
			for range rounds {
				p.Verbose(true)
				p.PrintRaw("block")
				p.Info("line", "worker", i)
			}
		})
	}
	wg.Wait()

	// Every PrintRaw block arrives whole, with no other caller's line inside it.
	if got := strings.Count(buf.String(), "\n block\n\n"); got != workers*rounds {
		t.Errorf("expected %d whole PrintRaw blocks, got %d", workers*rounds, got)
	}
}
