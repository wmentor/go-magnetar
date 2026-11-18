package epub_test

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/codec/epub"
)

func createTestEPUB(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir := t.TempDir()
	epubPath := filepath.Join(tmpDir, "test.epub")

	f, err := os.Create(epubPath)
	if err != nil {
		t.Fatalf("failed to create test epub file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("failed to write content to zip entry %s: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return epubPath
}

func TestCodec_ReadFile_StandardEPUB(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`,
		"OEBPS/content.opf": `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="ch1" href="chapter1.xhtml" media-type="application/xhtml+xml"/>
    <item id="ch2" href="chapter2.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
    <itemref idref="ch2"/>
  </spine>
</package>`,
		"OEBPS/chapter1.xhtml": `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 1</title></head>
<body>
  <h1>Chapter 1: The Beginning</h1>
  <p>Hello &amp; welcome to the <b>EPUB</b> reader test.</p>
</body>
</html>`,
		"OEBPS/chapter2.xhtml": `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Chapter 2</title></head>
<body>
  <h2>Chapter 2: Continuation</h2>
  <p>This is the second chapter with &quot;special quotes&quot;.</p>
</body>
</html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "# Chapter 1: The Beginning") {
		t.Errorf("expected content to contain Markdown heading 1, got:\n%s", content)
	}
	if !strings.Contains(content, "Hello & welcome to the **EPUB** reader test.") {
		t.Errorf("expected content to contain Markdown bold text, got:\n%s", content)
	}
	if !strings.Contains(content, "## Chapter 2: Continuation") {
		t.Errorf("expected content to contain Markdown heading 2, got:\n%s", content)
	}
	if !strings.Contains(content, `with "special quotes".`) {
		t.Errorf("expected content to contain Chapter 2 text, got:\n%s", content)
	}
}

func TestCodec_ReadFile_SpineOrdering(t *testing.T) {
	t.Parallel()

	// Ensure spine order (ch2 before ch1) is respected
	files := map[string]string{
		"META-INF/container.xml": `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf"/></rootfiles>
</container>`,
		"content.opf": `<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="first" href="first.xhtml" media-type="application/xhtml+xml"/>
    <item id="second" href="second.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="second"/>
    <itemref idref="first"/>
  </spine>
</package>`,
		"first.xhtml":  `<html><body><p>FIRST PARAGRAPH</p></body></html>`,
		"second.xhtml": `<html><body><p>SECOND PARAGRAPH</p></body></html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	idxSecond := strings.Index(content, "SECOND PARAGRAPH")
	idxFirst := strings.Index(content, "FIRST PARAGRAPH")

	if idxSecond == -1 || idxFirst == -1 {
		t.Fatalf("expected both paragraphs, got:\n%s", content)
	}
	if idxSecond >= idxFirst {
		t.Errorf("expected SECOND PARAGRAPH to appear before FIRST PARAGRAPH due to spine order, got:\n%s", content)
	}
}

func TestCodec_ReadFile_NoContainerFallback(t *testing.T) {
	t.Parallel()

	// Missing container.xml but package.opf present
	files := map[string]string{
		"book/package.opf": `<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="c1" href="intro.html" media-type="text/html"/>
  </manifest>
  <spine>
    <itemref idref="c1"/>
  </spine>
</package>`,
		"book/intro.html": `<html><body><p>Fallback OPF Intro</p></body></html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "Fallback OPF Intro") {
		t.Errorf("expected content to contain intro, got:\n%s", content)
	}
}

func TestCodec_ReadFile_DirectHTMLFallback(t *testing.T) {
	t.Parallel()

	// Missing container.xml and opf, but has html files
	files := map[string]string{
		"page1.html": `<html><body><p>Direct HTML Page 1</p></body></html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(content, "Direct HTML Page 1") {
		t.Errorf("expected content to contain direct HTML text, got:\n%s", content)
	}
}

func TestCodec_ReadFile_EPUB3(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="EPUB/package.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`,
		"EPUB/package.opf": `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="pub-id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="pub-id">urn:uuid:12345</dc:identifier>
    <dc:title>EPUB 3 Sample</dc:title>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="sec1" href="section1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="nav"/>
    <itemref idref="sec1"/>
  </spine>
</package>`,
		"EPUB/nav.xhtml": `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<head><title>Table of Contents</title></head>
<body>
  <nav epub:type="toc">
    <h1>Table of Contents</h1>
    <ol>
      <li><a href="section1.xhtml">Section 1</a></li>
    </ol>
  </nav>
</body>
</html>`,
		"EPUB/section1.xhtml": `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title>Section 1</title></head>
<body>
  <article>
    <header>
      <h1>Modern EPUB 3 Article</h1>
    </header>
    <section>
      <p>This is a test of EPUB 3.0 HTML5 structure support.</p>
      <aside><p>Note: Semantic markup is preserved.</p></aside>
    </section>
    <footer>
      <p>Article footer text</p>
    </footer>
  </article>
</body>
</html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Table of Contents",
		"Section 1",
		"Modern EPUB 3 Article",
		"This is a test of EPUB 3.0 HTML5 structure support.",
		"Note: Semantic markup is preserved.",
		"Article footer text",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected content to contain %q, got:\n%s", snippet, content)
		}
	}
}

func TestCodec_ReadFile_InvalidFile(t *testing.T) {
	t.Parallel()

	c := &epub.Codec{}

	// Non-existent file
	_, err := c.ReadFile("non_existent_file.epub")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}

	// Empty zip file with no content
	emptyZip := createTestEPUB(t, map[string]string{})
	_, err = c.ReadFile(emptyZip)
	if err == nil {
		t.Error("expected error for empty zip, got nil")
	}
	if !errors.Is(err, epub.ErrInvalidFileFormat) {
		t.Errorf("expected ErrInvalidFileFormat, got: %v", err)
	}
}

func TestCodec_ReadFile_MarkdownFormatting(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"META-INF/container.xml": `<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles><rootfile full-path="content.opf"/></rootfiles>
</container>`,
		"content.opf": `<package xmlns="http://www.idpf.org/2007/opf" version="2.0">
  <manifest>
    <item id="md" href="formatted.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="md"/>
  </spine>
</package>`,
		"formatted.xhtml": `<!DOCTYPE html>
<html>
<body>
  <h1>Rich Formatting</h1>
  <p>Here is <em>italic</em> and <strong>bold</strong> and <code>inline_code</code>.</p>
  <ul>
    <li>Item A</li>
    <li>Item B</li>
  </ul>
  <blockquote><p>A wise quote.</p></blockquote>
</body>
</html>`,
	}

	epubPath := createTestEPUB(t, files)

	c := &epub.Codec{}
	content, err := c.ReadFile(epubPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedParts := []string{
		"# Rich Formatting",
		"*italic*",
		"**bold**",
		"`inline_code`",
		"- Item A",
		"- Item B",
		"> A wise quote.",
	}

	for _, part := range expectedParts {
		if !strings.Contains(content, part) {
			t.Errorf("expected content to contain %q, got:\n%s", part, content)
		}
	}
}
