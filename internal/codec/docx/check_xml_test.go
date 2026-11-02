package docx

import (
	"archive/zip"
	"io"
	"os"
	"testing"
)

func TestCheckXML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-xml-*.docx")
	if err != nil {
		t.Fatalf("create temp file error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	codec := &Codec{}
	content := "Таинственный"
	err = codec.WriteFile(tmpFile.Name(), content)
	if err != nil {
		t.Fatalf("write file error: %v", err)
	}

	// Open and read document.xml
	r, err := zip.OpenReader(tmpFile.Name())
	if err != nil {
		t.Fatalf("open docx error: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open XML error: %v", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("read XML error: %v", err)
			}
			t.Logf("document.xml:\n%s", string(data))
		}
	}
}
