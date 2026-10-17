package render

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Options holds conversion options.
type Options struct {
	PreserveHTML      bool
	WarnOnUnsupported bool
	Verbose           bool
}

// Result holds conversion result with warnings.
type Result struct {
	Output   string
	Warnings []string
}

// Language mapping from Markdown to JIRA.
var languageMap = map[string]string{
	"js":         "javascript",
	"javascript": "javascript",
	"ts":         "typescript",
	"typescript": "typescript",
	"py":         "python",
	"python":     "python",
	"rb":         "ruby",
	"ruby":       "ruby",
	"sh":         "bash",
	"bash":       "bash",
	"shell":      "bash",
	"json":       "json",
	"xml":        "xml",
	"html":       "html",
	"css":        "css",
	"sql":        "sql",
	"java":       "java",
	"go":         "go",
	"golang":     "go",
	"rust":       "rust",
	"c":          "cpp",
	"cpp":        "cpp",
	"c++":        "cpp",
	"yaml":       "yaml",
	"yml":        "yaml",
	"php":        "php",
	"swift":      "swift",
	"kotlin":     "kotlin",
	"scala":      "scala",
	"r":          "r",
	"perl":       "perl",
	"groovy":     "groovy",
	"powershell": "powershell",
	"ps1":        "powershell",
	"dockerfile": "dockerfile",
	"makefile":   "makefile",
	"markdown":   "none",
	"md":         "none",
	"text":       "none",
	"txt":        "none",
	"plaintext":  "none",
}

// JIRARenderer renders Markdown AST to JIRA markup.
type JIRARenderer struct {
	source   []byte
	warnings []string
	options  Options
	// Track list nesting
	listStack []ast.Node
	// Track if we're in a tight list
	inTightList bool
	// Track blockquote content
	inBlockquote   bool
	blockquoteText strings.Builder
}

// NewJIRARenderer creates a new JIRA renderer.
func NewJIRARenderer(source []byte, opts Options) *JIRARenderer {
	return &JIRARenderer{
		source:    source,
		options:   opts,
		listStack: make([]ast.Node, 0),
		warnings:  make([]string, 0),
	}
}

// Render renders the AST to JIRA markup.
func (r *JIRARenderer) Render(doc ast.Node) string {
	var buf strings.Builder
	r.renderNode(&buf, doc, true)
	return buf.String()
}

// GetWarnings returns any warnings generated during rendering.
func (r *JIRARenderer) GetWarnings() []string {
	return r.warnings
}

// addWarning adds a warning message.
func (r *JIRARenderer) addWarning(msg string) {
	r.warnings = append(r.warnings, msg)
}

// renderNode renders a single node and its children.
func (r *JIRARenderer) renderNode(buf *strings.Builder, node ast.Node, entering bool) {
	switch n := node.(type) {
	case *ast.Document:
		r.renderChildren(buf, n)
	case *ast.Heading:
		r.renderHeading(buf, n, entering)
	case *ast.Paragraph:
		r.renderParagraph(buf, n, entering)
	case *ast.Text:
		r.renderText(buf, n, entering)
	case *ast.String:
		r.renderString(buf, n, entering)
	case *ast.Emphasis:
		r.renderEmphasis(buf, n, entering)
	case *ast.CodeSpan:
		r.renderCodeSpan(buf, n, entering)
	case *ast.FencedCodeBlock:
		r.renderFencedCodeBlock(buf, n, entering)
	case *ast.CodeBlock:
		r.renderCodeBlock(buf, n, entering)
	case *ast.Link:
		r.renderLink(buf, n, entering)
	case *ast.AutoLink:
		r.renderAutoLink(buf, n, entering)
	case *ast.Image:
		r.renderImage(buf, n, entering)
	case *ast.List:
		r.renderList(buf, n, entering)
	case *ast.ListItem:
		r.renderListItem(buf, n, entering)
	case *ast.ThematicBreak:
		r.renderThematicBreak(buf, n, entering)
	case *ast.Blockquote:
		r.renderBlockquote(buf, n, entering)
	case *ast.HTMLBlock:
		r.renderHTMLBlock(buf, n, entering)
	case *ast.RawHTML:
		r.renderRawHTML(buf, n, entering)
	case *ast.TextBlock:
		r.renderTextBlock(buf, n, entering)
	case *east.Table:
		r.renderTable(buf, n, entering)
	case *east.TableHeader:
		r.renderTableHeader(buf, n, entering)
	case *east.TableRow:
		r.renderTableRow(buf, n, entering)
	case *east.TableCell:
		r.renderTableCell(buf, n, entering)
	case *east.Strikethrough:
		r.renderStrikethrough(buf, n, entering)
	case *east.TaskCheckBox:
		r.renderTaskCheckBox(buf, n, entering)
	default:
		// For unknown nodes, try to render children
		if entering {
			r.renderChildren(buf, node)
		}
	}
}

// renderChildren renders all children of a node.
func (r *JIRARenderer) renderChildren(buf *strings.Builder, node ast.Node) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		r.walk(buf, child)
	}
}

// walk walks the AST and renders nodes.
func (r *JIRARenderer) walk(buf *strings.Builder, node ast.Node) {
	r.renderNode(buf, node, true)
	if !r.isLeafNode(node) && !r.skipChildren(node) {
		r.renderChildren(buf, node)
	}
	r.renderNode(buf, node, false)
}

// isLeafNode returns true if the node is a leaf (no children to render separately).
func (r *JIRARenderer) isLeafNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.Text, *ast.String, *ast.CodeSpan, *ast.FencedCodeBlock,
		*ast.CodeBlock, *ast.ThematicBreak, *ast.HTMLBlock, *ast.RawHTML,
		*east.TaskCheckBox:
		return true
	}
	return false
}

// skipChildren returns true if we handle children ourselves.
func (r *JIRARenderer) skipChildren(node ast.Node) bool {
	switch node.(type) {
	case *ast.Link, *ast.Image, *ast.AutoLink:
		return true
	}
	return false
}

// renderHeading renders a heading.
func (r *JIRARenderer) renderHeading(buf *strings.Builder, n *ast.Heading, entering bool) {
	if entering {
		fmt.Fprintf(buf, "h%d. ", n.Level)
	} else {
		buf.WriteString("\n\n")
	}
}

// renderParagraph renders a paragraph.
func (r *JIRARenderer) renderParagraph(buf *strings.Builder, _ *ast.Paragraph, entering bool) {
	if !entering {
		// Check if we're in a tight list
		if !r.inTightList || len(r.listStack) == 0 {
			buf.WriteString("\n\n")
		}
	}
}

// renderText renders text content.
func (r *JIRARenderer) renderText(buf *strings.Builder, n *ast.Text, entering bool) {
	if entering {
		text := string(n.Segment.Value(r.source))
		// Escape JIRA special characters in text
		text = r.escapeJIRAText(text)
		buf.WriteString(text)
		if n.HardLineBreak() {
			buf.WriteString("\\\\\n")
		} else if n.SoftLineBreak() {
			buf.WriteString("\n")
		}
	}
}

// renderString renders a string node.
func (r *JIRARenderer) renderString(buf *strings.Builder, n *ast.String, entering bool) {
	if entering {
		text := string(n.Value)
		text = r.escapeJIRAText(text)
		buf.WriteString(text)
	}
}

// escapeJIRAText escapes special characters for JIRA.
func (r *JIRARenderer) escapeJIRAText(text string) string {
	// Characters that have special meaning in JIRA and need escaping
	// We need to be careful not to double-escape or break formatting
	// Only escape when the character would be interpreted as formatting
	return text
}

// renderEmphasis renders emphasis (bold/italic).
func (r *JIRARenderer) renderEmphasis(buf *strings.Builder, n *ast.Emphasis, _ bool) {
	switch n.Level {
	case 1:
		// Single emphasis = italic
		buf.WriteString("_")
	case 2:
		// Double emphasis = bold
		buf.WriteString("*")
	}
	// Note: goldmark parses ***text*** as nested Emphasis nodes (level 2 containing level 1),
	// not as a single level 3 node. The nesting handles bold+italic automatically.
}

// renderStrikethrough renders strikethrough text.
func (r *JIRARenderer) renderStrikethrough(buf *strings.Builder, _ *east.Strikethrough, _ bool) {
	buf.WriteString("-")
}

// renderCodeSpan renders inline code.
func (r *JIRARenderer) renderCodeSpan(buf *strings.Builder, n *ast.CodeSpan, entering bool) {
	if entering {
		buf.WriteString("{{")
		// Get the code content
		for range n.ChildCount() {
			segment := n.Text(r.source)
			buf.Write(segment)
			break
		}
		buf.WriteString("}}")
	}
}

// renderFencedCodeBlock renders a fenced code block.
func (r *JIRARenderer) renderFencedCodeBlock(buf *strings.Builder, n *ast.FencedCodeBlock, entering bool) {
	if entering {
		lang := string(n.Language(r.source))
		lang = strings.TrimSpace(lang)

		// Map language to JIRA equivalent
		jiraLang := r.mapLanguage(lang)

		if jiraLang != "" && jiraLang != "none" {
			fmt.Fprintf(buf, "{code:%s}\n", jiraLang)
		} else {
			buf.WriteString("{code}\n")
		}

		// Get code content
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(r.source))
		}

		buf.WriteString("{code}\n\n")
	}
}

// renderCodeBlock renders an indented code block.
func (r *JIRARenderer) renderCodeBlock(buf *strings.Builder, n *ast.CodeBlock, entering bool) {
	if entering {
		buf.WriteString("{code}\n")

		// Get code content
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(r.source))
		}

		buf.WriteString("{code}\n\n")
	}
}

// mapLanguage maps Markdown language identifiers to JIRA equivalents.
func (r *JIRARenderer) mapLanguage(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if mapped, ok := languageMap[lang]; ok {
		return mapped
	}
	// Return the language as-is if no mapping exists
	return lang
}

// renderLink renders a link.
func (r *JIRARenderer) renderLink(buf *strings.Builder, n *ast.Link, entering bool) {
	if entering {
		// Get link text
		var linkText strings.Builder
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderLinkContent(&linkText, child)
		}

		url := string(n.Destination)
		text := linkText.String()

		if text == "" || text == url {
			fmt.Fprintf(buf, "[%s]", url)
		} else {
			fmt.Fprintf(buf, "[%s|%s]", text, url)
		}
	}
}

// renderLinkContent renders content inside a link.
func (r *JIRARenderer) renderLinkContent(buf *strings.Builder, node ast.Node) {
	switch n := node.(type) {
	case *ast.Text:
		buf.Write(n.Segment.Value(r.source))
	case *ast.String:
		buf.Write(n.Value)
	case *ast.CodeSpan:
		buf.WriteString("{{")
		buf.Write(n.Text(r.source))
		buf.WriteString("}}")
	case *ast.Emphasis:
		if n.Level == 1 {
			buf.WriteString("_")
		} else {
			buf.WriteString("*")
		}
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderLinkContent(buf, child)
		}
		if n.Level == 1 {
			buf.WriteString("_")
		} else {
			buf.WriteString("*")
		}
	default:
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			r.renderLinkContent(buf, child)
		}
	}
}

// renderAutoLink renders an autolink.
func (r *JIRARenderer) renderAutoLink(buf *strings.Builder, n *ast.AutoLink, entering bool) {
	if entering {
		url := string(n.URL(r.source))
		fmt.Fprintf(buf, "[%s]", url)
	}
}

// renderImage renders an image.
func (r *JIRARenderer) renderImage(buf *strings.Builder, n *ast.Image, entering bool) {
	if entering {
		url := string(n.Destination)
		// JIRA image syntax: !url! or !url|alt=text!
		alt := r.getImageAlt(n)
		if alt != "" {
			fmt.Fprintf(buf, "!%s|alt=%s!", url, alt)
		} else {
			fmt.Fprintf(buf, "!%s!", url)
		}
	}
}

// getImageAlt gets the alt text from an image node.
func (r *JIRARenderer) getImageAlt(n *ast.Image) string {
	var alt strings.Builder
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			alt.Write(text.Segment.Value(r.source))
		}
	}
	return alt.String()
}

// renderList renders a list.
func (r *JIRARenderer) renderList(buf *strings.Builder, n *ast.List, entering bool) {
	if entering {
		// If we're already in a list (nested list), add a newline before
		if len(r.listStack) > 0 {
			buf.WriteString("\n")
		}
		r.listStack = append(r.listStack, n)
		r.inTightList = n.IsTight
	} else {
		if len(r.listStack) > 0 {
			r.listStack = r.listStack[:len(r.listStack)-1]
		}
		if len(r.listStack) == 0 {
			r.inTightList = false
			buf.WriteString("\n")
		}
	}
}

// renderListItem renders a list item.
func (r *JIRARenderer) renderListItem(buf *strings.Builder, _ *ast.ListItem, entering bool) {
	if entering {
		// Build the list prefix based on nesting
		prefix := r.buildListPrefix()
		buf.WriteString(prefix)
		buf.WriteString(" ")
	} else {
		buf.WriteString("\n")
	}
}

// buildListPrefix builds the appropriate list prefix based on nesting.
func (r *JIRARenderer) buildListPrefix() string {
	var prefix strings.Builder
	for _, node := range r.listStack {
		if list, ok := node.(*ast.List); ok {
			if list.IsOrdered() {
				prefix.WriteString("#")
			} else {
				prefix.WriteString("*")
			}
		}
	}
	return prefix.String()
}

// renderThematicBreak renders a horizontal rule.
func (r *JIRARenderer) renderThematicBreak(buf *strings.Builder, _ *ast.ThematicBreak, entering bool) {
	if entering {
		buf.WriteString("----\n\n")
	}
}

// renderBlockquote renders a blockquote.
func (r *JIRARenderer) renderBlockquote(buf *strings.Builder, _ *ast.Blockquote, entering bool) {
	if entering {
		r.inBlockquote = true
		r.blockquoteText.Reset()
		buf.WriteString("{quote}\n")
	} else {
		r.inBlockquote = false
		buf.WriteString("{quote}\n\n")
	}
}

// renderHTMLBlock renders an HTML block.
func (r *JIRARenderer) renderHTMLBlock(buf *strings.Builder, n *ast.HTMLBlock, entering bool) {
	if entering {
		if r.options.PreserveHTML {
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				buf.Write(line.Value(r.source))
			}
		} else {
			// Try to convert common HTML tags
			lines := n.Lines()
			var html strings.Builder
			for i := 0; i < lines.Len(); i++ {
				line := lines.At(i)
				html.Write(line.Value(r.source))
			}
			converted := r.convertHTML(html.String())
			buf.WriteString(converted)
		}
		if r.options.WarnOnUnsupported {
			r.addWarning("HTML block found - converted with best effort")
		}
	}
}

// renderRawHTML renders inline HTML.
func (r *JIRARenderer) renderRawHTML(buf *strings.Builder, n *ast.RawHTML, entering bool) {
	if entering {
		segments := n.Segments
		var html strings.Builder
		for i := 0; i < segments.Len(); i++ {
			segment := segments.At(i)
			html.Write(segment.Value(r.source))
		}
		htmlStr := html.String()
		converted := r.convertHTML(htmlStr)
		// Debug logging
		_ = converted
		buf.WriteString(converted)
	}
}

// convertHTML converts common HTML to JIRA markup.
func (r *JIRARenderer) convertHTML(html string) string {
	// Convert <sup> to ^text^
	supRe := regexp.MustCompile(`<sup>([^<]*)</sup>`)
	html = supRe.ReplaceAllString(html, "^${1}^")

	// Convert <sub> to ~text~
	subRe := regexp.MustCompile(`<sub>([^<]*)</sub>`)
	html = subRe.ReplaceAllString(html, "~${1}~")

	// Convert <br> and <br/> to \\\\
	brRe := regexp.MustCompile(`<br\s*/?>`)
	html = brRe.ReplaceAllString(html, "\\\\\n")

	// Convert <strong> and <b> to *text*
	strongRe := regexp.MustCompile(`<(?:strong|b)>([^<]*)</(?:strong|b)>`)
	html = strongRe.ReplaceAllString(html, "*${1}*")

	// Convert <em> and <i> to _text_
	emRe := regexp.MustCompile(`<(?:em|i)>([^<]*)</(?:em|i)>`)
	html = emRe.ReplaceAllString(html, "_${1}_")

	// Convert <code> to {{text}}
	codeRe := regexp.MustCompile(`<code>([^<]*)</code>`)
	html = codeRe.ReplaceAllString(html, "{{${1}}}")

	// Convert <del> and <s> to -text-
	delRe := regexp.MustCompile(`<(?:del|s)>([^<]*)</(?:del|s)>`)
	html = delRe.ReplaceAllString(html, "-${1}-")

	// Convert <u> to +text+
	uRe := regexp.MustCompile(`<u>([^<]*)</u>`)
	html = uRe.ReplaceAllString(html, "+${1}+")

	// Strip remaining HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	html = tagRe.ReplaceAllString(html, "")

	return html
}

// renderTextBlock renders a text block.
func (r *JIRARenderer) renderTextBlock(buf *strings.Builder, n *ast.TextBlock, entering bool) {
	// Text blocks are typically children of list items in tight lists
	// We don't add extra newlines for them
}

// renderTable renders a table.
func (r *JIRARenderer) renderTable(buf *strings.Builder, _ *east.Table, entering bool) {
	if !entering {
		buf.WriteString("\n")
	}
}

// renderTableHeader renders a table header row.
func (r *JIRARenderer) renderTableHeader(buf *strings.Builder, _ *east.TableHeader, entering bool) {
	if !entering {
		buf.WriteString("\n")
	}
}

// renderTableRow renders a table row.
func (r *JIRARenderer) renderTableRow(buf *strings.Builder, _ *east.TableRow, entering bool) {
	if !entering {
		buf.WriteString("\n")
	}
}

// renderTableCell renders a table cell.
func (r *JIRARenderer) renderTableCell(buf *strings.Builder, n *east.TableCell, entering bool) {
	if entering {
		// Check if this is a header cell
		parent := n.Parent()
		_, isHeader := parent.(*east.TableHeader)

		if isHeader {
			buf.WriteString("||")
		} else {
			buf.WriteString("|")
		}
	} else {
		// Check if this is the last cell in the row
		if n.NextSibling() == nil {
			parent := n.Parent()
			_, isHeader := parent.(*east.TableHeader)
			if isHeader {
				buf.WriteString("||")
			} else {
				buf.WriteString("|")
			}
		}
	}
}

// renderTaskCheckBox renders a task checkbox.
func (r *JIRARenderer) renderTaskCheckBox(buf *strings.Builder, n *east.TaskCheckBox, entering bool) {
	if entering {
		if n.IsChecked {
			buf.WriteString("(/) ")
		} else {
			buf.WriteString("( ) ")
		}
	}
}

// Convert converts Markdown to JIRA markup.
func Convert(markdown string) string {
	result, _ := ConvertWithOptions(markdown, Options{})
	return result.Output
}

// ConvertWithOptions converts Markdown to JIRA markup with options.
func ConvertWithOptions(markdown string, opts Options) (Result, error) {
	// Create goldmark parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown (tables, strikethrough, etc.)
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// Parse the markdown
	source := []byte(markdown)
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	// Create renderer and render
	renderer := NewJIRARenderer(source, opts)
	output := renderer.Render(doc)

	// Clean up output
	output = cleanOutput(output)

	return Result{
		Output:   output,
		Warnings: renderer.GetWarnings(),
	}, nil
}

// cleanOutput cleans up the output.
func cleanOutput(output string) string {
	// Remove excessive blank lines (more than 2 consecutive)
	blankLineRe := regexp.MustCompile(`\n{3,}`)
	output = blankLineRe.ReplaceAllString(output, "\n\n")

	// Trim trailing whitespace from each line
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	output = strings.Join(lines, "\n")

	// Trim leading and trailing whitespace from the whole output
	output = strings.TrimSpace(output)

	return output
}

// Converter provides the conversion API.
type Converter struct {
	options Options
}

// NewConverter creates a new converter with default options.
func NewConverter() *Converter {
	return &Converter{}
}

// NewConverterWithOptions creates a new converter with specified options.
func NewConverterWithOptions(opts Options) *Converter {
	return &Converter{options: opts}
}

// Convert converts Markdown to JIRA markup.
func (c *Converter) Convert(markdown string) string {
	result, _ := ConvertWithOptions(markdown, c.options)
	return result.Output
}

// ConvertWithWarnings converts Markdown and returns warnings.
func (c *Converter) ConvertWithWarnings(markdown string) (string, []string) {
	result, _ := ConvertWithOptions(markdown, c.options)
	return result.Output, result.Warnings
}

// ConvertBytes converts Markdown bytes to JIRA markup bytes.
func (c *Converter) ConvertBytes(markdown []byte) []byte {
	result, _ := ConvertWithOptions(string(markdown), c.options)
	return []byte(result.Output)
}

// ConvertReader converts from a reader to a writer.
func (c *Converter) ConvertReader(r io.Reader, w io.Writer) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	result, _ := ConvertWithOptions(string(input), c.options)
	_, err = w.Write([]byte(result.Output))
	return err
}

// MustConvert converts Markdown to JIRA markup, panicking on error.
func MustConvert(markdown string) string {
	return Convert(markdown)
}
