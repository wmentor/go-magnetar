package epub

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html"
)

var ErrInvalidFileFormat = errors.New("invalid epub file format")

type Codec struct{}

type containerXML struct {
	XMLName   xml.Name `xml:"container"`
	Rootfiles []struct {
		FullPath  string `xml:"full-path,attr"`
		MediaType string `xml:"media-type,attr"`
	} `xml:"rootfiles>rootfile"`
}

type opfPackage struct {
	XMLName  xml.Name `xml:"package"`
	Manifest struct {
		Items []struct {
			ID        string `xml:"id,attr"`
			Href      string `xml:"href,attr"`
			MediaType string `xml:"media-type,attr"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		Itemrefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func (c *Codec) ReadFile(filename string) (string, error) {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	// Map all files in the archive by name for quick lookup.
	fileMap := make(map[string]*zip.File, len(zipReader.File))
	for _, f := range zipReader.File {
		fileMap[f.Name] = f
		cleaned := path.Clean(f.Name)
		if _, exists := fileMap[cleaned]; !exists {
			fileMap[cleaned] = f
		}
	}

	// 1. Locate the OPF rootfile path.
	opfPath := findOPFPath(zipReader.File, fileMap)
	if opfPath == "" {
		// Fallback: look for any .xhtml/.html files if no OPF file exists.
		return readDirectHTMLFiles(zipReader.File)
	}

	// 2. Read and parse the OPF document.
	opfFile := findFileInMap(fileMap, zipReader.File, opfPath)
	if opfFile == nil {
		return readDirectHTMLFiles(zipReader.File)
	}

	opfData, err := readZipFile(opfFile)
	if err != nil {
		return "", err
	}

	var opf opfPackage
	if err := xml.Unmarshal(opfData, &opf); err != nil {
		return readDirectHTMLFiles(zipReader.File)
	}

	opfDir := path.Dir(opfPath)
	if opfDir == "." {
		opfDir = ""
	}

	// Build manifest item map: id -> resolved path
	manifestMap := make(map[string]string, len(opf.Manifest.Items))
	var allManifestContentPaths []string

	for _, item := range opf.Manifest.Items {
		href := item.Href
		if idx := strings.Index(href, "#"); idx != -1 {
			href = href[:idx]
		}
		if unescaped, err := url.PathUnescape(href); err == nil {
			href = unescaped
		}
		resolved := path.Clean(path.Join(opfDir, href))
		manifestMap[item.ID] = resolved

		mediaType := strings.ToLower(item.MediaType)
		ext := strings.ToLower(path.Ext(resolved))
		if isContentDocument(mediaType, ext) {
			allManifestContentPaths = append(allManifestContentPaths, resolved)
		}
	}

	// Collect chapter paths in spine reading order.
	var chapterPaths []string
	seen := make(map[string]bool)

	for _, ref := range opf.Spine.Itemrefs {
		if resolved, ok := manifestMap[ref.IDRef]; ok && resolved != "" {
			if !seen[resolved] {
				seen[resolved] = true
				chapterPaths = append(chapterPaths, resolved)
			}
		}
	}

	// If spine produced no chapters, fall back to all manifest content documents.
	if len(chapterPaths) == 0 {
		for _, p := range allManifestContentPaths {
			if !seen[p] {
				seen[p] = true
				chapterPaths = append(chapterPaths, p)
			}
		}
	}

	// If still no chapters, fall back to scanning any html/xhtml in archive.
	if len(chapterPaths) == 0 {
		return readDirectHTMLFiles(zipReader.File)
	}

	// Read each chapter and extract plain text.
	var sections []string
	for _, chapPath := range chapterPaths {
		zf := findFileInMap(fileMap, zipReader.File, chapPath)
		if zf == nil {
			continue
		}

		data, err := readZipFile(zf)
		if err != nil {
			continue
		}

		text := extractTextFromHTML(data)
		if text != "" {
			sections = append(sections, text)
		}
	}

	if len(sections) == 0 {
		return "", ErrInvalidFileFormat
	}

	return strings.Join(sections, "\n\n"), nil
}

func isContentDocument(mediaType, ext string) bool {
	if mediaType == "application/xhtml+xml" || mediaType == "text/html" || mediaType == "application/html" {
		return true
	}
	return ext == ".xhtml" || ext == ".html" || ext == ".htm"
}

func findOPFPath(files []*zip.File, fileMap map[string]*zip.File) string {
	// Look for META-INF/container.xml
	containerFile := findFileInMap(fileMap, files, "META-INF/container.xml")
	if containerFile != nil {
		if data, err := readZipFile(containerFile); err == nil {
			var container containerXML
			if err := xml.Unmarshal(data, &container); err == nil {
				for _, rf := range container.Rootfiles {
					if rf.FullPath != "" {
						return rf.FullPath
					}
				}
			}
		}
	}

	// Fallback: search for any file ending in .opf
	for _, f := range files {
		if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
			return f.Name
		}
	}

	return ""
}

func readDirectHTMLFiles(files []*zip.File) (string, error) {
	var htmlFiles []*zip.File
	for _, f := range files {
		ext := strings.ToLower(path.Ext(f.Name))
		if ext == ".xhtml" || ext == ".html" || ext == ".htm" {
			htmlFiles = append(htmlFiles, f)
		}
	}

	if len(htmlFiles) == 0 {
		return "", ErrInvalidFileFormat
	}

	sort.Slice(htmlFiles, func(i, j int) bool {
		return htmlFiles[i].Name < htmlFiles[j].Name
	})

	var sections []string
	for _, f := range htmlFiles {
		data, err := readZipFile(f)
		if err != nil {
			continue
		}
		text := extractTextFromHTML(data)
		if text != "" {
			sections = append(sections, text)
		}
	}

	if len(sections) == 0 {
		return "", ErrInvalidFileFormat
	}

	return strings.Join(sections, "\n\n"), nil
}

func findFileInMap(fileMap map[string]*zip.File, files []*zip.File, target string) *zip.File {
	if f, ok := fileMap[target]; ok {
		return f
	}
	cleaned := path.Clean(target)
	if f, ok := fileMap[cleaned]; ok {
		return f
	}
	// Case-insensitive fallback
	lowerTarget := strings.ToLower(cleaned)
	for _, f := range files {
		if strings.ToLower(path.Clean(f.Name)) == lowerTarget {
			return f
		}
	}
	return nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func extractTextFromHTML(data []byte) string {
	doc, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return ""
	}
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style" || n.Data == "head") {
			return
		}
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				buf.WriteString(text)
				buf.WriteString(" ")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "br", "tr":
				buf.WriteString("\n\n")
			}
		}
	}
	walk(doc)
	return cleanParagraphs(buf.String())
}

func cleanParagraphs(text string) string {
	lines := strings.Split(text, "\n")
	var cleanLines []string
	lastEmpty := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !lastEmpty && len(cleanLines) > 0 {
				cleanLines = append(cleanLines, "")
				lastEmpty = true
			}
		} else {
			cleanLines = append(cleanLines, trimmed)
			lastEmpty = false
		}
	}
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}
