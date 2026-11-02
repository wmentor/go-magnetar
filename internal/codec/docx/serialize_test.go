package docx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"strings"
	"testing"
)

func TestNewSerializer(t *testing.T) {
	t.Parallel()

	codec := &Codec{}

	markdown := `# Heading 1

Some text here.

## Heading 2

* List item 1
* List item 2

| Column 1 | Column 2 |
|----------|----------|
| Cell 1   | Cell 2   |
`

	tmpFile, err := os.CreateTemp("", "test-new-serializer-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	tmpFilename := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpFilename)

	err = codec.WriteFile(tmpFilename, markdown)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Read the file and check structure
	readFile, err := os.Open(tmpFilename)
	if err != nil {
		t.Fatalf("open file error: %v", err)
	}
	defer readFile.Close()

	fileInfo, err := readFile.Stat()
	if err != nil {
		t.Fatalf("stat file error: %v", err)
	}

	zipReader, err := zip.NewReader(readFile, fileInfo.Size())
	if err != nil {
		t.Fatalf("create zip reader error: %v", err)
	}

	// Check that document.xml exists and has correct structure
	var documentXML []byte
	for _, zf := range zipReader.File {
		if zf.Name == "word/document.xml" {
			rc, err := zf.Open()
			if err != nil {
				t.Fatalf("open document.xml error: %v", err)
			}
			defer rc.Close()

			documentXML, err = io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read document.xml error: %v", err)
			}
			break
		}
	}

	if documentXML == nil {
		t.Fatal("document.xml not found")
	}

	docStr := string(documentXML)

	// Check that sectPr is at the end (before </w:body>)
	sectPrIdx := strings.Index(docStr, "<w:sectPr>")
	bodyEndIdx := strings.Index(docStr, "</w:body>")
	if sectPrIdx == -1 || bodyEndIdx == -1 {
		t.Fatal("sectPr or body end not found")
	}
	if sectPrIdx > bodyEndIdx {
		t.Error("sectPr should be before </w:body>")
	}

	// Check that table has tblGrid with gridCol
	if !strings.Contains(docStr, "<w:tblGrid>") {
		t.Error("table should have tblGrid")
	}
	if !strings.Contains(docStr, "<w:gridCol") {
		t.Error("tblGrid should have gridCol")
	}

	// Check that table has tblStyle
	if !strings.Contains(docStr, "<w:tblStyle") {
		t.Error("table should have tblStyle")
	}

	// Check that headings have correct styles
	if !strings.Contains(docStr, `w:val="Heading1"`) {
		t.Error("should have Heading1 style")
	}
	if !strings.Contains(docStr, `w:val="Heading2"`) {
		t.Error("should have Heading2 style")
	}

	t.Logf("Document XML:\n%s", docStr)
}

func TestSerializeDocument(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	doc.AddParagraph("Test paragraph", "")
	doc.AddParagraph("Heading 1", "Heading1")
	doc.AddListItem("Item 1", 0, 1)
	doc.AddTable([][]string{
		{"A", "B"},
		{"1", "2"},
	})

	xmlData, err := MarshalDocument(doc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	xmlStr := string(xmlData)

	// Check XML declaration
	if !strings.HasPrefix(xmlStr, "<?xml") {
		t.Error("should start with XML declaration")
	}

	// Check namespaces
	if !strings.Contains(xmlStr, `xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"`) {
		t.Error("should have w namespace")
	}

	// Check structure
	if !strings.Contains(xmlStr, "<w:document") {
		t.Error("should have w:document")
	}
	if !strings.Contains(xmlStr, "<w:body>") {
		t.Error("should have w:body")
	}
	if !strings.Contains(xmlStr, "<w:p>") {
		t.Error("should have paragraphs")
	}
	if !strings.Contains(xmlStr, "<w:tbl>") {
		t.Error("should have table")
	}

	// Check sectPr is at the end
	sectPrIdx := strings.Index(xmlStr, "<w:sectPr>")
	bodyEndIdx := strings.Index(xmlStr, "</w:body>")
	if sectPrIdx == -1 || bodyEndIdx == -1 {
		t.Fatal("sectPr or body end not found")
	}
	if sectPrIdx > bodyEndIdx {
		t.Error("sectPr should be before </w:body>")
	}

	t.Logf("Generated XML:\n%s", xmlStr)
}

func TestMarshalBodyOrder(t *testing.T) {
	t.Parallel()

	body := &Body{
		Items: []any{
			&Paragraph{
				Content: []ParagraphContent{Run{Text: TextContent{Value: "Test"}}},
			},
		},
		SectPr: SectionProperties{
			PgSz: PageSize{W: "11906", H: "16838"},
		},
	}

	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	err := body.MarshalXML(enc, xml.StartElement{Name: xml.Name{Local: "w:body"}})
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	result := buf.String()

	// Check that paragraph comes before sectPr
	pIdx := strings.Index(result, "<w:p>")
	sectPrIdx := strings.Index(result, "<w:sectPr>")
	if pIdx == -1 || sectPrIdx == -1 {
		t.Fatal("paragraph or sectPr not found")
	}
	if pIdx > sectPrIdx {
		t.Error("paragraph should come before sectPr")
	}
}
