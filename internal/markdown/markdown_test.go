package markdown_test

import (
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/markdown"
)

func TestRowsToTable_EmptyRows(t *testing.T) {
	t.Parallel()

	result := markdown.RowsToTable(nil)
	if result != "" {
		t.Errorf("expected empty string for nil rows, got: %q", result)
	}

	result = markdown.RowsToTable([][]string{})
	if result != "" {
		t.Errorf("expected empty string for empty rows, got: %q", result)
	}
}

func TestRowsToTable_EmptyFirstRow(t *testing.T) {
	t.Parallel()

	rows := [][]string{{}}
	result := markdown.RowsToTable(rows)
	if result != "" {
		t.Errorf("expected empty string for empty first row, got: %q", result)
	}
}

func TestRowsToTable_SingleRow(t *testing.T) {
	t.Parallel()

	rows := [][]string{{"Name", "Age", "City"}}
	expected := "| Name | Age | City |\n| --- | --- | --- |\n"

	result := markdown.RowsToTable(rows)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRowsToTable_MultipleRows(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"Name", "Age", "City"},
		{"John", "30", "New York"},
		{"Jane", "25", "Los Angeles"},
	}
	expected := "| Name | Age | City |\n| --- | --- | --- |\n| John | 30 | New York |\n| Jane | 25 | Los Angeles |\n"

	result := markdown.RowsToTable(rows)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRowsToTable_RowsWithDifferentColumnCounts(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"A", "B", "C"},
		{"X", "Y"},
		{"P", "Q", "R", "S"},
	}
	expected := "| A | B | C |\n| --- | --- | --- |\n| X | Y |  |\n| P | Q | R |\n"

	result := markdown.RowsToTable(rows)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRowsToTable_StringsWithSpaces(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"  Name  ", "  Age  "},
		{"  John Doe  ", "  30  "},
	}
	expected := "| Name | Age |\n| --- | --- |\n| John Doe | 30 |\n"

	result := markdown.RowsToTable(rows)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRowsToTable_StringsWithPipeCharacter(t *testing.T) {
	t.Parallel()

	rows := [][]string{
		{"Name", "Value"},
		{"A|B", "X|Y"},
	}
	expected := "| Name | Value |\n| --- | --- |\n| A\\|B | X\\|Y |\n"

	result := markdown.RowsToTable(rows)
	if result != expected {
		t.Errorf("expected:\n%q\ngot:\n%q", expected, result)
	}
}

func TestRowsToTable_LargeTable(t *testing.T) {
	t.Parallel()

	rows := make([][]string, 100)
	for i := range rows {
		rows[i] = []string{string(rune('A' + i%26)), string(rune('0' + i%10)), "Value"}
	}

	result := markdown.RowsToTable(rows)
	if result == "" {
		t.Error("expected non-empty result for large table")
	}

	lines := strings.Count(result, "\n")
	expectedLines := 100 + 1
	if lines != expectedLines {
		t.Errorf("expected %d lines, got %d", expectedLines, lines)
	}
}
