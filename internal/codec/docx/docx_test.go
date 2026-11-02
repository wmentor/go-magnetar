package docx_test

import (
	"os"
	"strings"
	"testing"

	docxlib "github.com/fumiama/go-docx"

	"github.com/wmentor/go-magnetar/internal/codec/docx"
)

func TestReadFile(t *testing.T) {
	t.Parallel()

	filename := "./testdata/1.docx"

	codec := &docx.Codec{}

	data, err := codec.ReadFile(filename)
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	expect, err := os.ReadFile("testdata/1.md")
	if err != nil {
		t.Fatalf("read file %q error: %v", filename, err)
	}

	if string(expect) != data {
		t.Fatalf("invalid result")
	}
}

func TestWriteFile(t *testing.T) {
	t.Parallel()

	// Test markdown content
	markdownContent := `Таинственный производитель чипов

С 2022 г. компания старается скрыть все свои проекты.

List:

* 1231312
* 121231313123
  * fwqerq
  * ewrqqwerweqr
    * fdsasdfasd fsas 
* 12312312312

Лист2:

* 123
  * 123112
  * 1231231231
* 22222
* авыфафы авыффыва

|  :----: | :----: | :----: |
| col1 | col2 | col3 |
| row11 | row12 | row13 |
| row21 | row22 | row23 |
| row31 | row32 | row33 |

Источник считает это следствием ошибок государственного планирования, поскольку большинство таких заводов построены с привлечением бюджетных средств: «За эти деньги нужно было построить две независимые мегафабрики, к которым бы обращались все российские заказчики. Бизнес мегафабрик должен заниматься только услугами по корпусированию, чтобы не создавать конфликты интересов с заказчиками».
`

	// Create a temporary file for testing
	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Create codec and write file
	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Verify the file exists and has content
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("stat file error: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}

	// Read back the file and verify it can be parsed
	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	t.Logf("Read data:\n%s", readData)

	// Verify content was preserved (rough check - compare key strings)
	if !strings.Contains(readData, "Таинственный производитель чипов") {
		t.Error("missing expected text in round-trip")
	}
	if !strings.Contains(readData, "col1") {
		t.Error("missing table content in round-trip")
	}
	if !strings.Contains(readData, "1231312") {
		t.Error("missing list content in round-trip")
	}
}

func TestWriteFile_SimpleParagraphs(t *testing.T) {
	t.Parallel()

	markdownContent := `First paragraph

Second paragraph

Third paragraph
`

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	if !strings.Contains(readData, "First paragraph") {
		t.Error("missing first paragraph in round-trip")
	}
	if !strings.Contains(readData, "Second paragraph") {
		t.Error("missing second paragraph in round-trip")
	}
}

func TestWriteFile_Headings(t *testing.T) {
	t.Parallel()

	markdownContent := `# Heading 1

## Heading 2

### Heading 3
`

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	if !strings.Contains(readData, "Heading 1") {
		t.Error("missing heading 1 in round-trip")
	}
	if !strings.Contains(readData, "Heading 2") {
		t.Error("missing heading 2 in round-trip")
	}
	if !strings.Contains(readData, "Heading 3") {
		t.Error("missing heading 3 in round-trip")
	}
}

func TestWriteFile_EmptyContent(t *testing.T) {
	t.Parallel()

	markdownContent := ""

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write empty content error: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(tmpfile.Name())
	if err != nil {
		t.Fatalf("stat file error: %v", err)
	}
	// Even empty markdown should create a minimal valid docx
	if info.Size() == 0 {
		t.Fatal("output file is empty")
	}
}

func TestWriteFile_CodeBlocks(t *testing.T) {
	t.Parallel()

	markdownContent := `Some text before

` + "```go" + `
func hello() {
    fmt.Println("Hello, World!")
}
` + "```" + `

Text after code block
`

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file with code block error: %v", err)
	}

	// Read back
	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	// Print read data for debugging
	t.Logf("Read back data: %q", readData)
	t.Logf("Count of 'Some text before': %d", strings.Count(readData, "Some text before"))
	t.Logf("Count of 'func hello': %d", strings.Count(readData, "func hello"))

	// Verify content
	if !strings.Contains(readData, "func hello") {
		t.Error("missing code content in round-trip")
	}
	if !strings.Contains(readData, "fmt.Println") {
		t.Error("missing code content in round-trip")
	}
}

func TestWriteFile_SimpleText(t *testing.T) {
	t.Parallel()

	// Simple test to see if text duplication happens
	markdownContent := "Hello\nWorld\n"

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	// First write
	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Debug: Read the docx file and check paragraphs before reading back
	readFile, err := os.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("open file error: %v", err)
	}
	defer readFile.Close()

	fileinfo, err := readFile.Stat()
	if err != nil {
		t.Fatalf("stat file error: %v", err)
	}

	doc, err := docxlib.Parse(readFile, fileinfo.Size())
	if err != nil {
		t.Fatalf("parse docx error: %v", err)
	}

	t.Logf("Number of items after write: %d", len(doc.Document.Body.Items))
	for i, it := range doc.Document.Body.Items {
		if para, ok := it.(*docxlib.Paragraph); ok {
			t.Logf("Paragraph %d children: %d", i, len(para.Children))
			for j, child := range para.Children {
				if run, ok := child.(*docxlib.Run); ok {
					t.Logf("  Run %d children: %d", j, len(run.Children))
					for k, rc := range run.Children {
						if text, ok := rc.(*docxlib.Text); ok {
							t.Logf("    Text %d: %q", k, text.Text)
						}
					}
				}
			}
		}
	}

	// Now read back using ReadFile
	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	t.Logf("Read back data: %q", readData)
	t.Logf("Count of 'Hello': %d", strings.Count(readData, "Hello"))
	t.Logf("Count of 'World': %d", strings.Count(readData, "World"))

	// Verify each text appears only once
	if strings.Count(readData, "Hello") != 1 {
		t.Errorf("Expected 'Hello' to appear once, got %d times", strings.Count(readData, "Hello"))
	}
	if strings.Count(readData, "World") != 1 {
		t.Errorf("Expected 'World' to appear once, got %d times", strings.Count(readData, "World"))
	}
}

func TestWriteFile_InlineCode(t *testing.T) {
	t.Parallel()

	markdownContent := `This is inline ` + "`code`" + ` in a sentence.

Another ` + "`example`" + ` here.
`

	tmpfile, err := os.CreateTemp("", "test-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	codec := &docx.Codec{}

	err = codec.WriteFile(tmpfile.Name(), markdownContent)
	if err != nil {
		t.Fatalf("write file with inline code error: %v", err)
	}

	readData, err := codec.ReadFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("read back file error: %v", err)
	}

	if !strings.Contains(readData, "code") {
		t.Error("missing inline code content in round-trip")
	}
	if !strings.Contains(readData, "example") {
		t.Error("missing inline code content in round-trip")
	}
}
