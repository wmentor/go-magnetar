package docx

import (
	"encoding/xml"
	"fmt"
	"strconv"
)

// XML namespaces.
const (
	nsW  = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	nsR  = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	nsWP = "http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"
)

// Document represents the root document.xml structure.
type Document struct {
	XMLName     xml.Name        `xml:"w:document"`
	XMLNSW      string          `xml:"xmlns:w,attr"`
	XMLNSR      string          `xml:"xmlns:r,attr"`
	XMLNSWP     string          `xml:"xmlns:wp,attr"`
	Body        Body            `xml:"w:body"`
	Hyperlinks  []HyperlinkInfo // Track hyperlinks for relationships
	hyperlinkID int             // Counter for generating unique hyperlink IDs
}

// HyperlinkInfo stores information about a hyperlink for relationship generation.
type HyperlinkInfo struct {
	ID  string // rId for the relationship
	URL string // Target URL
}

// Body represents w:body.
type Body struct {
	Items  []any // Paragraph or Table
	SectPr SectionProperties
}

// SectionProperties represents w:sectPr (page settings).
type SectionProperties struct {
	XMLName xml.Name `xml:"w:sectPr"`
	PgSz    PageSize `xml:"w:pgSz"`
}

// PageSize represents w:pgSz (A4 = 11906 x 16838 twips).
type PageSize struct {
	XMLName xml.Name `xml:"w:pgSz"`
	W       string   `xml:"w:w,attr"`
	H       string   `xml:"w:h,attr"`
}

// Paragraph represents w:p.
type Paragraph struct {
	XMLName    xml.Name             `xml:"w:p"`
	Properties *ParagraphProperties `xml:"w:pPr,omitempty"`
	Content    []ParagraphContent   `xml:",any"`
}

// ParagraphContent is an interface for paragraph content (Run or Hyperlink).
type ParagraphContent interface {
	isParagraphContent()
}

// Run represents w:r.
type Run struct {
	XMLName    xml.Name       `xml:"w:r"`
	Properties *RunProperties `xml:"w:rPr,omitempty"`
	Text       TextContent    `xml:"w:t,omitempty"`
	Break      *LineBreak     `xml:"w:br,omitempty"`
}

// TextContent represents w:t with xml:space attribute for preserving whitespace.
type TextContent struct {
	Space string `xml:"xml:space,attr,omitempty"`
	Value string `xml:",chardata"`
}

// MarshalXML implements custom marshaling for TextContent.
func (t TextContent) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if t.Value == "" {
		return nil
	}

	// Always preserve whitespace to handle leading/trailing spaces
	type Alias TextContent
	return e.EncodeElement(struct {
		Alias
		Space string `xml:"xml:space,attr"`
	}{
		Alias: Alias(t),
		Space: "preserve",
	}, start)
}

func (r Run) isParagraphContent() {}

// Hyperlink represents w:hyperlink.
type Hyperlink struct {
	XMLName xml.Name `xml:"w:hyperlink"`
	ID      string   `xml:"r:id,attr"`
	Runs    []Run    `xml:"w:r"`
}

func (h Hyperlink) isParagraphContent() {}

// createHyperlinkRun creates a Run with hyperlink styling.
func createHyperlinkRun(text string, bold, italic bool) Run {
	run := createRun(text, bold, italic)
	// Add hyperlink character style
	if run.Properties == nil {
		run.Properties = &RunProperties{}
	}
	run.Properties.RStyle = &RunStyle{Val: "Hyperlink"}
	return run
}

// ParseMarkdownText parses markdown text and returns a slice of TextRun with formatting.
func ParseMarkdownText(text string) []TextRun {
	var runs []TextRun
	var current TextRun
	inBold := false
	inItalic := false

	// Use range to properly handle UTF-8 characters
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		// Check for link [text](url)
		if runes[i] == '[' {
			// Find closing ]
			closeBracket := -1
			for j := i + 1; j < len(runes); j++ {
				if runes[j] == ']' {
					closeBracket = j
					break
				}
			}

			// Check if followed by (url)
			if closeBracket > 0 && closeBracket+1 < len(runes) && runes[closeBracket+1] == '(' {
				closeParen := -1
				for j := closeBracket + 2; j < len(runes); j++ {
					if runes[j] == ')' {
						closeParen = j
						break
					}
				}

				if closeParen > 0 {
					// Save current text if any
					if current.Text != "" {
						current.Bold = inBold
						current.Italic = inItalic
						runs = append(runs, current)
						current = TextRun{}
					}

					// Extract link text and URL
					linkText := string(runes[i+1 : closeBracket])
					linkURL := string(runes[closeBracket+2 : closeParen])

					// Add as a link
					runs = append(runs, TextRun{
						Text: linkText,
						Link: linkURL,
					})

					i = closeParen + 1
					continue
				}
			}
		}

		// Check for bold (**text**)
		if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
			// Save current text if any
			if current.Text != "" {
				current.Bold = inBold
				current.Italic = inItalic
				runs = append(runs, current)
				current = TextRun{}
			}
			inBold = !inBold
			i += 2
			continue
		}

		// Check for italic (*text*)
		if runes[i] == '*' {
			// Save current text if any
			if current.Text != "" {
				current.Bold = inBold
				current.Italic = inItalic
				runs = append(runs, current)
				current = TextRun{}
			}
			inItalic = !inItalic
			i++
			continue
		}

		// Regular character
		current.Text += string(runes[i])
		i++
	}

	// Save last text if any
	if current.Text != "" {
		current.Bold = inBold
		current.Italic = inItalic
		runs = append(runs, current)
	}

	return runs
}

// ParagraphProperties represents w:pPr.
type ParagraphProperties struct {
	XMLName xml.Name        `xml:"w:pPr"`
	Style   *Style          `xml:"w:pStyle,omitempty"`
	NumPr   *NumberingProps `xml:"w:numPr,omitempty"`
	Jc      *Justification  `xml:"w:jc,omitempty"`
}

// Style represents w:pStyle.
type Style struct {
	XMLName xml.Name `xml:"w:pStyle"`
	Val     string   `xml:"w:val,attr"`
}

// Justification represents w:jc (paragraph alignment).
type Justification struct {
	XMLName xml.Name `xml:"w:jc"`
	Val     string   `xml:"w:val,attr"`
}

// NumberingProps represents w:numPr for lists.
type NumberingProps struct {
	XMLName xml.Name        `xml:"w:numPr"`
	Ilvl    *NumberingIlvl  `xml:"w:ilvl,omitempty"`
	NumID   *NumberingNumID `xml:"w:numId,omitempty"`
}

// NumberingIlvl represents w:ilvl element.
type NumberingIlvl struct {
	XMLName xml.Name `xml:"w:ilvl"`
	Val     string   `xml:"w:val,attr"`
}

// NumberingNumID represents w:numId element.
type NumberingNumID struct {
	XMLName xml.Name `xml:"w:numId"`
	Val     string   `xml:"w:val,attr"`
}

// LineBreak represents w:br (line break within a paragraph).
type LineBreak struct {
	XMLName xml.Name `xml:"w:br"`
}

// TextRun represents a text segment with formatting.
type TextRun struct {
	Text   string
	Bold   bool
	Italic bool
	Link   string // URL if this is a hyperlink
}

// RunProperties represents w:rPr.
type RunProperties struct {
	XMLName xml.Name  `xml:"w:rPr"`
	RStyle  *RunStyle `xml:"w:rStyle,omitempty"`
	RFonts  *RunFonts `xml:"w:rFonts,omitempty"`
	B       *string   `xml:"w:b,omitempty"`
	I       *string   `xml:"w:i,omitempty"`
}

// RunStyle represents w:rStyle.
type RunStyle struct {
	XMLName xml.Name `xml:"w:rStyle"`
	Val     string   `xml:"w:val,attr"`
}

// RunFonts represents w:rFonts for specifying font families.
type RunFonts struct {
	XMLName  xml.Name `xml:"w:rFonts"`
	ASCII    string   `xml:"w:ascii,attr,omitempty"`
	HAnsi    string   `xml:"w:hAnsi,attr,omitempty"`
	EastAsia string   `xml:"w:eastAsia,attr,omitempty"`
	CS       string   `xml:"w:cs,attr,omitempty"`
}

// Table represents w:tbl.
type Table struct {
	XMLName    xml.Name        `xml:"w:tbl"`
	Properties TableProperties `xml:"w:tblPr"`
	Grid       TableGrid       `xml:"w:tblGrid"`
	Rows       []TableRow      `xml:"w:tr"`
}

// TableProperties represents w:tblPr.
type TableProperties struct {
	XMLName xml.Name      `xml:"w:tblPr"`
	Style   *TableStyle   `xml:"w:tblStyle,omitempty"`
	Width   TableWidth    `xml:"w:tblW"`
	Borders *TableBorders `xml:"w:tblBorders,omitempty"`
}

// TableStyle represents w:tblStyle.
type TableStyle struct {
	XMLName xml.Name `xml:"w:tblStyle"`
	Val     string   `xml:"w:val,attr"`
}

// TableWidth represents w:tblW.
type TableWidth struct {
	XMLName xml.Name `xml:"w:tblW"`
	W       string   `xml:"w:w,attr"`
	Type    string   `xml:"w:type,attr"`
}

// TableBorders represents w:tblBorders.
type TableBorders struct {
	XMLName xml.Name      `xml:"w:tblBorders"`
	Top     BorderElement `xml:"w:top"`
	Left    BorderElement `xml:"w:left"`
	Bottom  BorderElement `xml:"w:bottom"`
	Right   BorderElement `xml:"w:right"`
	InsideH BorderElement `xml:"w:insideH"`
	InsideV BorderElement `xml:"w:insideV"`
}

// BorderElement represents a border (top, left, etc.).
type BorderElement struct {
	Val   string `xml:"w:val,attr"`
	Sz    string `xml:"w:sz,attr"`
	Color string `xml:"w:color,attr"`
}

// TableGrid represents w:tblGrid.
type TableGrid struct {
	XMLName xml.Name  `xml:"w:tblGrid"`
	Cols    []GridCol `xml:"w:gridCol"`
}

// GridCol represents w:gridCol.
type GridCol struct {
	XMLName xml.Name `xml:"w:gridCol"`
	W       string   `xml:"w:w,attr"`
}

// TableRow represents w:tr.
type TableRow struct {
	XMLName xml.Name    `xml:"w:tr"`
	Cells   []TableCell `xml:"w:tc"`
}

// TableCell represents w:tc.
type TableCell struct {
	XMLName    xml.Name        `xml:"w:tc"`
	Properties *CellProperties `xml:"w:tcPr,omitempty"`
	Paragraph  Paragraph       `xml:"w:p"`
}

// CellProperties represents w:tcPr.
type CellProperties struct {
	XMLName xml.Name  `xml:"w:tcPr"`
	Width   CellWidth `xml:"w:tcW"`
}

// CellWidth represents w:tcW.
type CellWidth struct {
	XMLName xml.Name `xml:"w:tcW"`
	W       string   `xml:"w:w,attr"`
	Type    string   `xml:"w:type,attr"`
}

// MarshalDocument serializes a Document to XML.
func MarshalDocument(doc *Document) ([]byte, error) {
	// Set namespaces.
	doc.XMLNSW = nsW
	doc.XMLNSR = nsR
	doc.XMLNSWP = nsWP

	// Marshal to XML.
	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal document: %w", err)
	}

	// Add XML declaration.
	return append([]byte(xml.Header), output...), nil
}

// GetHyperlinks returns all hyperlinks in the document.
func (d *Document) GetHyperlinks() []HyperlinkInfo {
	return d.Hyperlinks
}

// NewDocument creates a new Document with default settings (A4 page).
func NewDocument() *Document {
	return &Document{
		Body: Body{
			Items: make([]any, 0),
			SectPr: SectionProperties{
				PgSz: PageSize{
					W: "11906", // A4 width in twips
					H: "16838", // A4 height in twips
				},
			},
		},
	}
}

// createRun creates a Run with optional bold/italic formatting.
func createRun(text string, bold, italic bool) Run {
	run := Run{Text: TextContent{Value: text}}
	if bold || italic {
		run.Properties = &RunProperties{}
		if bold {
			b := "true"
			run.Properties.B = &b
		}
		if italic {
			i := "true"
			run.Properties.I = &i
		}
	}
	return run
}

// addTextRunsToParagraph adds text runs (with formatting and links) to a paragraph.
func (d *Document) addTextRunsToParagraph(p *Paragraph, textRuns []TextRun) {
	for _, tr := range textRuns {
		if tr.Link == "" {
			// Create regular run
			p.Content = append(p.Content, createRun(tr.Text, tr.Bold, tr.Italic))
			continue
		}

		// Create hyperlink
		d.hyperlinkID++
		rID := fmt.Sprintf("rId%d", d.hyperlinkID+100) // Offset to avoid conflicts

		// Store hyperlink info for relationship generation
		d.Hyperlinks = append(d.Hyperlinks, HyperlinkInfo{
			ID:  rID,
			URL: tr.Link,
		})

		// Create hyperlink with formatted text and hyperlink style
		run := createHyperlinkRun(tr.Text, tr.Bold, tr.Italic)
		hyperlink := Hyperlink{
			ID:   rID,
			Runs: []Run{run},
		}
		p.Content = append(p.Content, hyperlink)
	}
}

// AddParagraph adds a paragraph to the document.
func (d *Document) AddParagraph(text string, style string) {
	// Parse markdown for bold/italic/links
	textRuns := ParseMarkdownText(text)

	p := &Paragraph{
		Content: make([]ParagraphContent, 0, len(textRuns)),
	}

	if style != "" {
		p.Properties = &ParagraphProperties{
			Style: &Style{Val: style},
		}
	}

	d.addTextRunsToParagraph(p, textRuns)
	d.Body.Items = append(d.Body.Items, p)
}

// AddCodeBlock adds a code block as a single paragraph with line breaks.
func (d *Document) AddCodeBlock(lines []string) {
	if len(lines) == 0 {
		return
	}

	// Set left alignment for code blocks
	p := &Paragraph{
		Properties: &ParagraphProperties{
			Jc: &Justification{Val: "left"},
		},
		Content: make([]ParagraphContent, 0, len(lines)*2),
	}

	for i, line := range lines {
		// Add text run with preserved whitespace and monospace font
		run := Run{
			Text: TextContent{Value: line},
			Properties: &RunProperties{
				RFonts: &RunFonts{
					ASCII:    "Courier New",
					HAnsi:    "Courier New",
					EastAsia: "Courier New",
					CS:       "Courier New",
				},
			},
		}
		p.Content = append(p.Content, run)

		// Add line break after each line except the last one
		if i < len(lines)-1 {
			p.Content = append(p.Content, Run{Break: &LineBreak{}})
		}
	}

	d.Body.Items = append(d.Body.Items, p)
}

// AddListItem adds a list item to the document.
func (d *Document) AddListItem(text string, level int, numID int) {
	// Parse markdown for bold/italic/links
	textRuns := ParseMarkdownText(text)

	p := &Paragraph{
		Properties: &ParagraphProperties{
			NumPr: &NumberingProps{
				Ilvl:  &NumberingIlvl{Val: strconv.Itoa(level)},
				NumID: &NumberingNumID{Val: strconv.Itoa(numID)},
			},
		},
		Content: make([]ParagraphContent, 0, len(textRuns)),
	}

	d.addTextRunsToParagraph(p, textRuns)
	d.Body.Items = append(d.Body.Items, p)
}

// AddTable adds a table to the document.
func (d *Document) AddTable(rows [][]string) {
	if len(rows) == 0 {
		return
	}

	numCols := len(rows[0])

	// Create table grid with columns
	grid := TableGrid{
		Cols: make([]GridCol, numCols),
	}
	colWidth := 9000 / numCols // Distribute width evenly (9000 twips = ~15cm)
	for i := range grid.Cols {
		grid.Cols[i] = GridCol{W: strconv.Itoa(colWidth)}
	}

	// Create table rows
	tableRows := make([]TableRow, len(rows))
	for i, row := range rows {
		cells := make([]TableCell, numCols)
		for j := range cells {
			cellText := ""
			if j < len(row) {
				cellText = row[j]
			}
			cells[j] = TableCell{
				Properties: &CellProperties{
					Width: CellWidth{
						W:    strconv.Itoa(colWidth),
						Type: "dxa",
					},
				},
				Paragraph: Paragraph{
					Content: []ParagraphContent{
						Run{Text: TextContent{Value: cellText}},
					},
				},
			}
		}
		tableRows[i] = TableRow{Cells: cells}
	}

	table := &Table{
		Properties: TableProperties{
			Style: &TableStyle{Val: "TableGrid"},
			Width: TableWidth{
				W:    "0",
				Type: "auto",
			},
			Borders: &TableBorders{
				Top:     BorderElement{Val: "single", Sz: "4", Color: "auto"},
				Left:    BorderElement{Val: "single", Sz: "4", Color: "auto"},
				Bottom:  BorderElement{Val: "single", Sz: "4", Color: "auto"},
				Right:   BorderElement{Val: "single", Sz: "4", Color: "auto"},
				InsideH: BorderElement{Val: "single", Sz: "4", Color: "auto"},
				InsideV: BorderElement{Val: "single", Sz: "4", Color: "auto"},
			},
		},
		Grid: grid,
		Rows: tableRows,
	}

	d.Body.Items = append(d.Body.Items, table)
}

// MarshalBody serializes the body with sectPr at the end.
func (b *Body) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := e.EncodeToken(start); err != nil {
		return err
	}

	// Encode items first
	for _, item := range b.Items {
		if err := e.Encode(item); err != nil {
			return err
		}
	}

	// Encode sectPr last
	if err := e.Encode(b.SectPr); err != nil {
		return err
	}

	return e.EncodeToken(start.End())
}
