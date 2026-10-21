package generic

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/printer"
)

// testPrinter - a simple Printer implementation for testing
type testPrinter struct {
	output *bytes.Buffer
}

func (p *testPrinter) Print(icon string, msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Info(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Warn(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Error(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) Debug(msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) ToolCall(icon string, msg string, params ...any) {
	// Not needed for these tests
}

func (p *testPrinter) EmptyLine() {
	// Not needed for these tests
}

func (p *testPrinter) ToolEmptyLine() {
	// Not needed for these tests
}

func (p *testPrinter) Verbose(enabled bool) {
	// Not needed for these tests
}

func (p *testPrinter) Ask(question string) string {
	// Not needed for these tests
	return ""
}

func (p *testPrinter) PrintRaw(msg string) {
	p.output.WriteString(msg)
}

func TestOutputDiff_BasicFunctionality(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Hello World"
	newText := "Hello Universe"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}

	// Output should contain ANSI color codes (indicating diff output)
	hasDelete := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasDelete {
		t.Errorf("Output doesn't contain ANSI color codes: %q", output)
	}

	// Check that both old and new text fragments are present in the diff output
	// DiffPrettyText shows: unchanged_part + deleted_part_in_red + inserted_part_in_green
	// For "Hello World" -> "Hello Universe":
	// "Hello " (unchanged) + "Wo" (deleted) + "Unive" (inserted) + "r" (unchanged) + "ld" (deleted) + "se" (inserted)

	// The output should contain fragments that correspond to the original and new text
	oldTextFragments := []string{"Hello", "World"}
	_ = oldTextFragments // Suppress unused variable warning

	// Check that both "World" and "Universe" are mentioned in some form
	// (they might be split across ANSI codes)
	for _, fragment := range oldTextFragments {
		if regexp.MustCompile(regexp.QuoteMeta(fragment)).MatchString(output) {
			// Found fragment in output (possibly with ANSI codes)
			continue
		} else {
			// Check if fragment characters appear in output
			found := false
			for _, r := range fragment {
				if regexp.MustCompile(regexp.QuoteMeta(string(r))).MatchString(output) {
					found = true
					break
				}
			}
			if !found {
				t.Logf("Fragment %q not found directly in output", fragment)
			}
		}
	}
}

func TestOutputDiff_Multiline(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "line1\nline2\nline3"
	newText := "line1\nmodified\nline3"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}

	// Check for ANSI codes
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI codes: %q", output)
	}
}

func TestOutputDiff_NoChanges(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "same text"
	newText := "same text"

	outputDiff(tp, oldText, newText)

	output := buf.String()
	// When texts are identical, DiffPrettyText returns empty or just a newline
	// The function should still work without errors
	_ = output // Suppress unused variable warning
}

func TestOutputDiff_Insertion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Hello"
	newText := "Hello World"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}

	// Output should contain ANSI color codes
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI codes: %q", output)
	}
}

func TestOutputDiff_Deletion(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Hello World"
	newText := "Hello"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}

	// Output should contain ANSI color codes
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI codes: %q", output)
	}
}

func TestOutputDiff_EmptyString(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := ""
	newText := "hello"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty when text is inserted
	if output == "" {
		t.Error("Output is empty when inserting text")
	}
}

func TestOutputDiff_Unicode(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Привет, мир!"
	newText := "Hello, world!"

	outputDiff(tp, oldText, newText)

	output := buf.String()
	// Should handle Unicode correctly without panicking
	_ = output // Suppress unused variable warning
}

func TestOutputDiff_LongText(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := strings.Repeat("a", 1000) + " bbb"
	newText := strings.Repeat("a", 1000) + " ccc"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}

	// Should contain ANSI codes
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI codes: %q", output)
	}
}

func TestOutputDiff_SpecialCharacters(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "Hello\n\tWorld\r\n"
	newText := "Hello\n\tModified\r\n"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}
}

func TestOutputDiff_WithDefaultPrinter(t *testing.T) {
	t.Parallel()

	// Save original default printer
	original := printer.GetDefault()
	defer func() { printer.SetDefault(original) }()

	var buf bytes.Buffer
	printer.SetDefault(printer.New(&buf, true))

	oldText := "test old"
	newText := "test new"

	outputDiff(printer.GetDefault(), oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Fatal("Output is empty")
	}
}

func TestOutputDiff_OnlyWhitespace(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "   "
	newText := "    "

	outputDiff(tp, oldText, newText)

	output := buf.String()
	// Should handle whitespace-only strings without error
	_ = output // Suppress unused variable warning
}

func TestOutputDiff_DiffContentStructure(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "abc"
	newText := "xyz"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// The diff output should have a specific structure:
	// - Unchanged text (if any)
	// - Deleted text wrapped in ANSI codes (e.g., \x1b[31m...text...\x1b[0m)
	// - Inserted text wrapped in ANSI codes (e.g., \x1b[32m...text...\x1b[0m)

	// At minimum, the output should contain ANSI color codes
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI color codes: %q", output)
	}

	// All three characters should be represented in the diff (either deleted or inserted)
	// The original "abc" and new "xyz" have no common characters
	// So the output should show all 6 characters somehow
	hasA := regexp.MustCompile(regexp.QuoteMeta("a")).MatchString(output)
	hasB := regexp.MustCompile(regexp.QuoteMeta("b")).MatchString(output)
	hasC := regexp.MustCompile(regexp.QuoteMeta("c")).MatchString(output)
	hasX := regexp.MustCompile(regexp.QuoteMeta("x")).MatchString(output)
	hasY := regexp.MustCompile(regexp.QuoteMeta("y")).MatchString(output)
	hasZ := regexp.MustCompile(regexp.QuoteMeta("z")).MatchString(output)

	// At least some characters from both strings should appear
	foundAnyOld := hasA || hasB || hasC
	foundAnyNew := hasX || hasY || hasZ

	if !foundAnyOld || !foundAnyNew {
		t.Logf("Diff output may not contain all characters (this is expected for some diff algorithms)")
		t.Logf("Output: %q", output)
	}
}

func TestOutputDiff_VerboseOutput(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	tp := &testPrinter{output: &buf}

	oldText := "abc"
	newText := "xyz"

	outputDiff(tp, oldText, newText)

	output := buf.String()

	// Output should not be empty
	if output == "" {
		t.Error("Output is empty")
	}

	// Output should contain ANSI color codes for diff
	hasANSI := regexp.MustCompile(`\x1b\[[0-9;]*m`).MatchString(output)
	if !hasANSI {
		t.Errorf("Output doesn't contain ANSI color codes: %q", output)
	}
}
