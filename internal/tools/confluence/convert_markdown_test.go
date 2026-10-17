package confluence_test

import (
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/confluence"
)

func TestConvertMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			want:    "",
			wantErr: false,
		},
		{
			name:    "simple paragraph",
			input:   "Hello world",
			want:    "<p>Hello world</p>\n",
			wantErr: false,
		},
		{
			name:    "heading level 1",
			input:   "# Heading 1",
			want:    "<h1>Heading 1</h1>\n",
			wantErr: false,
		},
		{
			name:    "heading level 2",
			input:   "## Heading 2",
			want:    "<h2>Heading 2</h2>\n",
			wantErr: false,
		},
		{
			name:    "heading level 3",
			input:   "### Heading 3",
			want:    "<h3>Heading 3</h3>\n",
			wantErr: false,
		},
		{
			name:  "simple table",
			input: "| Column 1 | Column 2 | Column 3 |\n|----------|----------|----------|\n| Row 1    | Data 1   | Data 2   |\n| Row 2    | Data 3   | Data 4   |\n| Row 3    | Data 5   | Data 6   |",
			want: `<table>
<thead>
<tr>
<th>Column 1</th>
<th>Column 2</th>
<th>Column 3</th>
</tr>
</thead>
<tbody>
<tr>
<td>Row 1</td>
<td>Data 1</td>
<td>Data 2</td>
</tr>
<tr>
<td>Row 2</td>
<td>Data 3</td>
<td>Data 4</td>
</tr>
<tr>
<td>Row 3</td>
<td>Data 5</td>
<td>Data 6</td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:  "table with alignment",
			input: "| Left | Center | Right |\n|:-----|:------:|------:|\n| A    | B      | C     |\n| D    | E      | F     |",
			want: `<table>
<thead>
<tr>
<th style="text-align:left">Left</th>
<th style="text-align:center">Center</th>
<th style="text-align:right">Right</th>
</tr>
</thead>
<tbody>
<tr>
<td style="text-align:left">A</td>
<td style="text-align:center">B</td>
<td style="text-align:right">C</td>
</tr>
<tr>
<td style="text-align:left">D</td>
<td style="text-align:center">E</td>
<td style="text-align:right">F</td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:  "table with bold and italic",
			input: "| Name | Description |\n|------|-------------|\n| **Item 1** | *Description 1* |\n| Item 2 | Normal text |",
			want: `<table>
<thead>
<tr>
<th>Name</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td><strong>Item 1</strong></td>
<td><em>Description 1</em></td>
</tr>
<tr>
<td>Item 2</td>
<td>Normal text</td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:  "table with links",
			input: "| Name | URL |\n|------|-----|\n| Google | https://google.com |\n| Example | http://example.com |",
			want: `<table>
<thead>
<tr>
<th>Name</th>
<th>URL</th>
</tr>
</thead>
<tbody>
<tr>
<td>Google</td>
<td><a href="https://google.com">https://google.com</a></td>
</tr>
<tr>
<td>Example</td>
<td><a href="http://example.com">http://example.com</a></td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:  "mixed content with table",
			input: "# Title\n\nSome text before table.\n\n| Col 1 | Col 2 |\n|-------|-------|\n| A     | B     |\n\nSome text after table.\n",
			want: `<h1>Title</h1>
<p>Some text before table.</p>
<table>
<thead>
<tr>
<th>Col 1</th>
<th>Col 2</th>
</tr>
</thead>
<tbody>
<tr>
<td>A</td>
<td>B</td>
</tr>
</tbody>
</table>
<p>Some text after table.</p>
`,
			wantErr: false,
		},
		{
			name:  "typographer replacements in table",
			input: "| Quote | Ellipsis |\n|-------|----------|\n| \"Text\" | ...      |",
			want: `<table>
<thead>
<tr>
<th>Quote</th>
<th>Ellipsis</th>
</tr>
</thead>
<tbody>
<tr>
<td>&ldquo;Text&rdquo;</td>
<td>&hellip;</td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:    "invalid markdown syntax",
			input:   "| Column 1 | Column 2 |\n|----------|",
			want:    "<p>| Column 1 | Column 2 |\n|&mdash;&mdash;&mdash;-|</p>\n",
			wantErr: false,
		},
		{
			name:  "list with table",
			input: "1. First item\n2. Second item\n\n| A | B |\n|---|---|\n| C | D |",
			want: `<ol>
<li>First item</li>
<li>Second item</li>
</ol>
<table>
<thead>
<tr>
<th>A</th>
<th>B</th>
</tr>
</thead>
<tbody>
<tr>
<td>C</td>
<td>D</td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
		{
			name:  "nested formatting in table",
			input: "| Code | Bold | Italic |\n|------|------|--------|\n| `code` | **bold** | *italic* |",
			want: `<table>
<thead>
<tr>
<th>Code</th>
<th>Bold</th>
<th>Italic</th>
</tr>
</thead>
<tbody>
<tr>
<td><code>code</code></td>
<td><strong>bold</strong></td>
<td><em>italic</em></td>
</tr>
</tbody>
</table>
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := confluence.ConvertMarkdown(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertMarkdown(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if got != tt.want {
				t.Errorf("ConvertMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Verify the output is valid HTML (has opening and closing tags)
			if !tt.wantErr && tt.input != "" {
				// Check that some HTML tags are present
				if !strings.Contains(got, "<") || !strings.Contains(got, ">") {
					t.Errorf("ConvertMarkdown(%q) should produce valid HTML", tt.input)
				}
			}
		})
	}
}

func TestConvertMarkdown_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty table row",
			input: "| Col1 | Col2 |\n|------|------|\n|      |      |",
			want: `<table>
<thead>
<tr>
<th>Col1</th>
<th>Col2</th>
</tr>
</thead>
<tbody>
<tr>
<td></td>
<td></td>
</tr>
</tbody>
</table>
`,
		},
		{
			name:  "single column table",
			input: "| Single |\n|--------|\n| Data   |",
			want: `<table>
<thead>
<tr>
<th>Single</th>
</tr>
</thead>
<tbody>
<tr>
<td>Data</td>
</tr>
</tbody>
</table>
`,
		},
		{
			name:  "single row table",
			input: "| Col1 | Col2 | Col3 |\n|------|------|------|\n| A    | B    | C    |",
			want: `<table>
<thead>
<tr>
<th>Col1</th>
<th>Col2</th>
<th>Col3</th>
</tr>
</thead>
<tbody>
<tr>
<td>A</td>
<td>B</td>
<td>C</td>
</tr>
</tbody>
</table>
`,
		},
		{
			name:  "table with special characters",
			input: "| Special | Characters |\n|---------|------------|\n| <>&\"   | 'quote'    |",
			want: `<table>
<thead>
<tr>
<th>Special</th>
<th>Characters</th>
</tr>
</thead>
<tbody>
<tr>
<td>&lt;&gt;&amp;&quot;</td>
<td>&lsquo;quote&rsquo;</td>
</tr>
</tbody>
</table>
`,
		},
		{
			name:  "table with emojis",
			input: "| Emoji | Name |\n|-------|------|\n| 😀    | Smile |\n| 🔥    | Fire |",
			want: `<table>
<thead>
<tr>
<th>Emoji</th>
<th>Name</th>
</tr>
</thead>
<tbody>
<tr>
<td>😀</td>
<td>Smile</td>
</tr>
<tr>
<td>🔥</td>
<td>Fire</td>
</tr>
</tbody>
</table>
`,
		},
		{
			name:  "table with multiline content",
			input: "| Col1 | Col2 |\n|------|------|\n| Line 1<br>Line 2 | Normal |",
			want: `<table>
<thead>
<tr>
<th>Col1</th>
<th>Col2</th>
</tr>
</thead>
<tbody>
<tr>
<td>Line 1<!-- raw HTML omitted -->Line 2</td>
<td>Normal</td>
</tr>
</tbody>
</table>
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := confluence.ConvertMarkdown(tt.input)
			if err != nil {
				t.Errorf("ConvertMarkdown(%q) error = %v, want no error", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ConvertMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
