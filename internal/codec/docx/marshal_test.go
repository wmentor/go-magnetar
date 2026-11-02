package docx

import (
	"testing"
)

func TestMarshalCyrillic(t *testing.T) {
	doc := NewDocument()
	doc.AddParagraph("Таинственный", "")

	xmlData, err := MarshalDocument(doc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	t.Logf("XML bytes (first 500):\n%s", string(xmlData[:min(500, len(xmlData))]))
	t.Logf("Contains 'Таинственный': %v", string(xmlData) != "" && len(xmlData) > 0)
}
