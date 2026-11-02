package docx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fumiama/go-docx"
	"github.com/pkg/errors"
)

type Codec struct{}

func (c *Codec) ReadFile(filename string) (string, error) {
	readFile, err := os.Open(filename)
	if err != nil {
		return "", errors.Wrapf(err, "open %q error", filename)
	}
	defer readFile.Close()

	fileinfo, err := readFile.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "get file %q size error", filename)
	}
	size := fileinfo.Size()

	doc, err := docx.Parse(readFile, size)
	if err != nil {
		return "", errors.Wrapf(err, "decode docx file %q error", filename)
	}

	var buffer strings.Builder

	for _, it := range doc.Document.Body.Items {
		switch item := it.(type) {
		case *docx.Paragraph:
			// Get paragraph text using String() method
			// paraText := item.String()  // Uncomment for debugging
			if lvl, ok := getListLevel(item); ok {
				// Use String() method to get paragraph text, not fmt.Sprint
				str := strings.ReplaceAll(strings.TrimLeft(item.String(), " \t"), "\n", " ")
				str = strings.Repeat("  ", lvl) + "* " + str
				fmt.Fprintln(&buffer, str)
			} else {
				// Use String() method to get paragraph text
				fmt.Fprintln(&buffer, item.String())
				fmt.Fprintln(&buffer, "")
			}

		case *docx.Table:
			fmt.Fprintln(&buffer, it)
			fmt.Fprintln(&buffer, "")
		}
	}

	return buffer.String(), nil
}

func getListLevel(p *docx.Paragraph) (int, bool) {
	if p.Properties == nil || p.Properties.NumProperties == nil {
		return 0, false
	}

	if p.Properties.NumProperties.Ilvl != nil {
		level, err := strconv.Atoi(p.Properties.NumProperties.Ilvl.Val)
		if err != nil {
			return 0, true
		}
		return level, true
	}

	return 0, true
}

// WriteFile converts markdown text to .docx format and writes it to a file.
func (c *Codec) WriteFile(filename string, content string) error {
	// Create a new document using our custom serializer
	doc := NewDocument()

	// Split content into lines for processing
	lines := strings.Split(content, "\n")

	var currentText strings.Builder
	var currentStyle string
	var currentIsNumberedList bool
	listLevel := 0
	inTable := false
	var tableRows [][]string
	var currentRow []string
	inCodeBlock := false
	var codeBlockContent []string

	// Track list state for proper numbering
	listCount := 0  // Number of independent lists
	inList := false // Whether we're currently inside a list

	// Helper function to flush current paragraph
	flushParagraph := func() {
		text := strings.TrimSpace(currentText.String())
		if text != "" {
			if currentStyle == "list" {
				// Calculate numId based on list count and type
				// For bullet lists: odd numIds (1, 3, 5, ...)
				// For numbered lists: even numIds (2, 4, 6, ...)
				numID := listCount*2 - 1
				if currentIsNumberedList {
					numID = listCount * 2
				}
				doc.AddListItem(text, listLevel, numID)
			} else {
				doc.AddParagraph(text, currentStyle)
			}
		}
		currentText.Reset()
		currentStyle = ""
		currentIsNumberedList = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code block delimiters
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Start of code block
				flushParagraph()
				// Add empty line before code block
				doc.AddParagraph("", "")
				inCodeBlock = true
				codeBlockContent = nil
			} else {
				// End of code block
				inCodeBlock = false
				// Add the collected code content as a single paragraph with line breaks
				if len(codeBlockContent) > 0 {
					doc.AddCodeBlock(codeBlockContent)
				}
				// Add empty line after code block
				doc.AddParagraph("", "")
				codeBlockContent = nil
			}
			continue
		}

		// If inside code block, collect lines
		if inCodeBlock {
			codeBlockContent = append(codeBlockContent, line)
			continue
		}

		if trimmed == "" {
			// Empty line - finalize current elements
			if inTable {
				// Finalize table
				if len(tableRows) > 0 {
					doc.AddTable(tableRows)
				}
				inTable = false
				tableRows = nil
				currentRow = nil
			} else {
				flushParagraph()
			}
			// Track list state: if we were in a list and hit empty line, next list is new
			if inList {
				inList = false
			}
			continue
		}

		// Check for table markdown
		if strings.HasPrefix(trimmed, "|") {
			if !inTable {
				flushParagraph()
				inTable = true
				tableRows = make([][]string, 0)
			}
			c.parseTableRow(trimmed, &tableRows, &currentRow)
			continue
		}

		// If we were in a table and hit a non-table line, finalize table
		if inTable {
			doc.AddTable(tableRows)
			inTable = false
			tableRows = nil
			currentRow = nil
		}

		// Check for headings
		if strings.HasPrefix(trimmed, "#") {
			flushParagraph()
			level, text := c.parseHeading(trimmed)
			currentStyle = fmt.Sprintf("Heading%d", level)
			currentText.WriteString(text)
			flushParagraph()
			continue
		}

		// Check for list items
		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "1. ") || strings.HasPrefix(trimmed, "2. ") {
			flushParagraph()

			// If we weren't in a list or we were in a list but hit empty line before,
			// this is a new independent list
			if !inList {
				listCount++
				inList = true
			}

			level, text, isNumbered := c.parseListItem(line)
			listLevel = level
			currentStyle = "list"
			currentText.WriteString(text)
			// Store whether this is a numbered list for flushParagraph
			currentIsNumberedList = isNumbered
			flushParagraph()
			continue
		}

		// Regular paragraph text
		if currentText.Len() > 0 {
			currentText.WriteString(" ")
		}
		currentText.WriteString(trimmed)
	}

	// Finalize any remaining elements
	if inCodeBlock {
		// Close unclosed code block
		if len(codeBlockContent) > 0 {
			doc.AddCodeBlock(codeBlockContent)
		}
	} else if inTable {
		doc.AddTable(tableRows)
	} else {
		flushParagraph()
	}

	// Serialize document to XML
	documentXML, err := MarshalDocument(doc)
	if err != nil {
		return errors.Wrapf(err, "serialize document error")
	}

	// Get hyperlinks for relationship generation
	hyperlinks := doc.GetHyperlinks()

	// Write to file
	file, err := os.Create(filename)
	if err != nil {
		return errors.Wrapf(err, "create file %q error", filename)
	}
	defer file.Close()

	// Create ZIP archive
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add document.xml
	docWriter, err := zipWriter.Create("word/document.xml")
	if err != nil {
		return errors.Wrapf(err, "create document.xml in zip error")
	}
	if _, err := docWriter.Write(documentXML); err != nil {
		return errors.Wrapf(err, "write document.xml to zip error")
	}

	// Add template files (styles, theme, etc.)
	if err := c.addTemplateFiles(zipWriter); err != nil {
		return errors.Wrapf(err, "add template files error")
	}

	// Close zip writer to flush all data
	if err := zipWriter.Close(); err != nil {
		return errors.Wrapf(err, "close zip writer error")
	}

	// Inject required styles and numbering for Word compatibility
	if err := c.injectRequiredStyles(filename, listCount, hyperlinks); err != nil {
		return errors.Wrapf(err, "inject styles to file %q error", filename)
	}

	return nil
}

func (c *Codec) parseHeading(line string) (int, string) {
	// Count # symbols to determine heading level
	level := 0
	for i := 0; i < len(line) && line[i] == '#'; i++ {
		level++
	}

	// Extract heading text (remove # and leading space)
	text := strings.TrimSpace(strings.TrimLeft(line, "#"))

	return level, text
}

func (c *Codec) parseListItem(line string) (int, string, bool) {
	// Calculate level: each 2 spaces = 1 level
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " "))
	newLevel := 0
	if leadingSpaces >= 2 {
		newLevel = leadingSpaces / 2
	}

	// Determine if this is a numbered list (1., 2., etc.) or bullet list (*, -)
	isNumbered := false
	trimmedLine := strings.TrimSpace(line)
	if strings.HasPrefix(trimmedLine, "1. ") || strings.HasPrefix(trimmedLine, "2. ") ||
		strings.HasPrefix(trimmedLine, "3. ") || strings.HasPrefix(trimmedLine, "4. ") ||
		strings.HasPrefix(trimmedLine, "5. ") || strings.HasPrefix(trimmedLine, "6. ") ||
		strings.HasPrefix(trimmedLine, "7. ") || strings.HasPrefix(trimmedLine, "8. ") ||
		strings.HasPrefix(trimmedLine, "9. ") {
		isNumbered = true
	}

	// Extract item text (remove * or - or number. and space)
	text := trimmedLine
	if isNumbered {
		// Remove number and dot
		if idx := strings.Index(text, ". "); idx != -1 {
			text = text[idx+2:]
		}
	} else {
		// Remove bullet marker (- or *) followed by space
		if strings.HasPrefix(text, "- ") {
			text = text[2:]
		} else if strings.HasPrefix(text, "* ") {
			text = text[2:]
		}
	}

	return newLevel, text, isNumbered
}

func (c *Codec) parseTableRow(line string, tableRows *[][]string, currentRow *[]string) {
	// Parse table row: | col1 | col2 | col3 |
	// Skip header separator line (| --- | --- | --- |)
	trimmed := strings.TrimSpace(line)
	if strings.Contains(trimmed, "---") {
		return
	}

	// Split by | and extract cells
	parts := strings.Split(line, "|")
	// Skip if line doesn't have enough parts (at least 3: empty, content, empty)
	if len(parts) < 3 {
		return
	}
	for _, part := range parts {
		cell := strings.TrimSpace(part)
		if cell != "" && !strings.Contains(cell, "---") {
			*currentRow = append(*currentRow, cell)
		}
	}

	// Check if we have a complete row
	if len(*currentRow) > 0 && len(parts) > 0 && parts[len(parts)-1] == "" {
		*tableRows = append(*tableRows, *currentRow)
		*currentRow = nil
	}
}

// addTemplateFiles adds template files (styles, theme, etc.) to the zip archive.
func (c *Codec) addTemplateFiles(zipWriter *zip.Writer) error {
	// Create a temporary docx with go-docx to get template files
	tmpDoc := docx.New().WithDefaultTheme().WithA4Page()

	// Write to a buffer
	var buf bytes.Buffer
	tmpFile := &tempWriter{buf: &buf}
	if _, err := tmpDoc.WriteTo(tmpFile); err != nil {
		return errors.Wrap(err, "create template docx error")
	}

	// Read the buffer as zip
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return errors.Wrap(err, "read template zip error")
	}

	// Copy template files (except document.xml which we already wrote)
	for _, zf := range zipReader.File {
		if zf.Name == "word/document.xml" {
			continue // Skip, we already wrote our own
		}

		rc, err := zf.Open()
		if err != nil {
			return errors.Wrapf(err, "open %q in template error", zf.Name)
		}

		writer, err := zipWriter.Create(zf.Name)
		if err != nil {
			rc.Close()
			return errors.Wrapf(err, "create %q in zip error", zf.Name)
		}

		// Use LimitReader to prevent decompression bomb (100MB limit).
		const maxFileSize = 100 * 1024 * 1024
		limitedReader := io.LimitReader(rc, maxFileSize)
		if _, err := io.Copy(writer, limitedReader); err != nil {
			rc.Close()
			return errors.Wrapf(err, "copy %q to zip error", zf.Name)
		}
		rc.Close()
	}

	return nil
}

// tempWriter is a simple io.Writer that writes to a bytes.Buffer.
type tempWriter struct {
	buf *bytes.Buffer
}

func (t *tempWriter) Write(p []byte) (int, error) {
	return t.buf.Write(p)
}

// injectRequiredStyles adds missing styles, numbering, and settings for Word compatibility.
// The go-docx library's default template lacks:
// - Heading1-6 style definitions.
// - Table styles (TableGrid, TableNormal).
// - word/numbering.xml for proper list numbering.
// - word/settings.xml for document settings.
// - Proper relationships and content types.
func (c *Codec) injectRequiredStyles(filename string, listCount int, hyperlinks []HyperlinkInfo) error {
	// Open the docx file (which is a zip archive) for reading
	readFile, err := os.Open(filename)
	if err != nil {
		return errors.Wrapf(err, "open file %q error", filename)
	}
	defer readFile.Close()

	fileInfo, err := readFile.Stat()
	if err != nil {
		return errors.Wrapf(err, "stat file %q error", filename)
	}

	zipReader, err := zip.NewReader(readFile, fileInfo.Size())
	if err != nil {
		return errors.Wrapf(err, "create zip reader for %q error", filename)
	}

	// Create a temporary file for the modified docx
	tmpFile, err := os.CreateTemp("", "docx-styles-*.docx")
	if err != nil {
		return errors.Wrap(err, "create temp file error")
	}
	tmpFilename := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpFilename) // Clean up temp file
	}()

	zipWriter := zip.NewWriter(tmpFile)
	defer zipWriter.Close()

	// Track which files we've processed
	processedFiles := make(map[string]bool)

	// Process each file in the zip archive
	for _, zipFile := range zipReader.File {
		processedFiles[zipFile.Name] = true

		switch zipFile.Name {
		case "word/styles.xml":
			if err := c.processStylesXML(zipFile, zipWriter); err != nil {
				return err
			}
		case "word/_rels/document.xml.rels":
			if err := c.processRelsXML(zipFile, zipWriter, hyperlinks); err != nil {
				return err
			}
		case "[Content_Types].xml":
			if err := c.processContentTypesXML(zipFile, zipWriter); err != nil {
				return err
			}
		default:
			if err := c.copyFileAsIs(zipFile, zipWriter); err != nil {
				return err
			}
		}
	}

	// Add new files: numbering.xml and settings.xml
	if listCount > 0 {
		numberingXML := generateNumberingXML(listCount)
		writer, err := zipWriter.Create("word/numbering.xml")
		if err != nil {
			return errors.Wrap(err, "create word/numbering.xml in zip error")
		}
		if _, err := writer.Write(numberingXML); err != nil {
			return errors.Wrap(err, "write word/numbering.xml to zip error")
		}
	}

	settingsXML := generateSettingsXML()
	writer, err := zipWriter.Create("word/settings.xml")
	if err != nil {
		return errors.Wrap(err, "create word/settings.xml in zip error")
	}
	if _, err := writer.Write(settingsXML); err != nil {
		return errors.Wrap(err, "write word/settings.xml to zip error")
	}

	// Close writers before renaming
	if err := zipWriter.Close(); err != nil {
		return errors.Wrap(err, "close zip writer error")
	}
	if err := tmpFile.Close(); err != nil {
		return errors.Wrap(err, "close temp file error")
	}
	if err := readFile.Close(); err != nil {
		return errors.Wrap(err, "close original file error")
	}

	// Replace the original file with the modified one
	if err := os.Rename(tmpFilename, filename); err != nil {
		return errors.Wrapf(err, "rename temp file to %q error", filename)
	}

	return nil
}

// processStylesXML modifies styles.xml to add heading and table styles.
func (c *Codec) processStylesXML(zipFile *zip.File, zipWriter *zip.Writer) error {
	rc, err := zipFile.Open()
	if err != nil {
		return errors.Wrapf(err, "open %q in zip error", zipFile.Name)
	}

	stylesContent, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return errors.Wrapf(err, "read %q from zip error", zipFile.Name)
	}

	// Inject heading, table, and hyperlink styles
	modifiedStyles := injectHeadingStyles(stylesContent)
	modifiedStyles = injectTableStyles(modifiedStyles)
	modifiedStyles = injectHyperlinkStyles(modifiedStyles)

	writer, err := zipWriter.Create(zipFile.Name)
	if err != nil {
		return errors.Wrapf(err, "create %q in zip error", zipFile.Name)
	}

	if _, err := writer.Write(modifiedStyles); err != nil {
		return errors.Wrapf(err, "write %q to zip error", zipFile.Name)
	}
	return nil
}

// processRelsXML modifies relationships to include numbering, settings, and hyperlinks.
func (c *Codec) processRelsXML(zipFile *zip.File, zipWriter *zip.Writer, hyperlinks []HyperlinkInfo) error {
	rc, err := zipFile.Open()
	if err != nil {
		return errors.Wrapf(err, "open %q in zip error", zipFile.Name)
	}

	relsContent, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return errors.Wrapf(err, "read %q from zip error", zipFile.Name)
	}

	// Add numbering, settings, and hyperlink relationships
	modifiedRels := injectRelationships(relsContent, hyperlinks)

	writer, err := zipWriter.Create(zipFile.Name)
	if err != nil {
		return errors.Wrapf(err, "create %q in zip error", zipFile.Name)
	}

	if _, err := writer.Write(modifiedRels); err != nil {
		return errors.Wrapf(err, "write %q to zip error", zipFile.Name)
	}
	return nil
}

// processContentTypesXML modifies content types to include numbering and settings.
func (c *Codec) processContentTypesXML(zipFile *zip.File, zipWriter *zip.Writer) error {
	rc, err := zipFile.Open()
	if err != nil {
		return errors.Wrapf(err, "open %q in zip error", zipFile.Name)
	}

	contentTypesData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return errors.Wrapf(err, "read %q from zip error", zipFile.Name)
	}

	// Add numbering and settings content types
	modifiedContentTypes := injectContentTypes(contentTypesData)

	writer, err := zipWriter.Create(zipFile.Name)
	if err != nil {
		return errors.Wrapf(err, "create %q in zip error", zipFile.Name)
	}

	if _, err := writer.Write(modifiedContentTypes); err != nil {
		return errors.Wrapf(err, "write %q to zip error", zipFile.Name)
	}
	return nil
}

// copyFileAsIs copies a file from the source zip to the destination zip without modification.
func (c *Codec) copyFileAsIs(zipFile *zip.File, zipWriter *zip.Writer) error {
	rc, err := zipFile.Open()
	if err != nil {
		return errors.Wrapf(err, "open %q in zip error", zipFile.Name)
	}
	defer rc.Close()

	writer, err := zipWriter.Create(zipFile.Name)
	if err != nil {
		return errors.Wrapf(err, "create %q in zip error", zipFile.Name)
	}

	// Use LimitReader to prevent decompression bomb attacks (100MB limit per file)
	const maxFileSize = 100 * 1024 * 1024
	limitedReader := io.LimitReader(rc, maxFileSize)
	if _, err := io.Copy(writer, limitedReader); err != nil {
		return errors.Wrapf(err, "copy %q to zip error", zipFile.Name)
	}
	return nil
}

// injectHeadingStyles adds Heading1-6 style definitions to styles.xml content.
func injectHeadingStyles(stylesContent []byte) []byte {
	// Define heading styles XML
	headingStyles := `
    <w:style w:type="paragraph" w:styleId="Heading1">
      <w:name w:val="heading 1"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="480" w:after="0"/>
        <w:outlineLvl w:val="0"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="48"/>
        <w:szCs w:val="48"/>
      </w:rPr>
    </w:style>
    <w:style w:type="paragraph" w:styleId="Heading2">
      <w:name w:val="heading 2"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="360" w:after="0"/>
        <w:outlineLvl w:val="1"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="36"/>
        <w:szCs w:val="36"/>
      </w:rPr>
    </w:style>
    <w:style w:type="paragraph" w:styleId="Heading3">
      <w:name w:val="heading 3"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="288" w:after="0"/>
        <w:outlineLvl w:val="2"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="28"/>
        <w:szCs w:val="28"/>
      </w:rPr>
    </w:style>
    <w:style w:type="paragraph" w:styleId="Heading4">
      <w:name w:val="heading 4"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="240" w:after="0"/>
        <w:outlineLvl w:val="3"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="24"/>
        <w:szCs w:val="24"/>
      </w:rPr>
    </w:style>
    <w:style w:type="paragraph" w:styleId="Heading5">
      <w:name w:val="heading 5"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="240" w:after="0"/>
        <w:outlineLvl w:val="4"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="21"/>
        <w:szCs w:val="21"/>
      </w:rPr>
    </w:style>
    <w:style w:type="paragraph" w:styleId="Heading6">
      <w:name w:val="heading 6"/>
      <w:basedOn w:val="Normal"/>
      <w:next w:val="Normal"/>
      <w:qFormat/>
      <w:pPr>
        <w:keepNext/>
        <w:keepLines/>
        <w:spacing w:before="240" w:after="0"/>
        <w:outlineLvl w:val="5"/>
      </w:pPr>
      <w:rPr>
        <w:b/>
        <w:sz w:val="18"/>
        <w:szCs w:val="18"/>
      </w:rPr>
    </w:style>
`

	// Find the closing </w:styles> tag and insert heading styles before it
	closingTag := "</w:styles>"
	idx := bytes.Index(stylesContent, []byte(closingTag))
	if idx == -1 {
		// If closing tag not found, return original content
		return stylesContent
	}

	// Insert heading styles before the closing tag
	var result bytes.Buffer
	result.Write(stylesContent[:idx])
	result.WriteString(headingStyles)
	result.WriteString("\n")
	result.Write(stylesContent[idx:])

	return result.Bytes()
}
