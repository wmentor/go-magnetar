package docx

import (
	"bytes"
	"fmt"
	"strings"
)

// generateNumberingXML creates word/numbering.xml with support for multiple independent lists.
// Each list gets its own num reference to prevent continuous numbering across the document.
func generateNumberingXML(numLists int) []byte {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:numbering xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" `)
	buf.WriteString(`xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
`)

	// Create one abstractNum for bullet lists
	buf.WriteString(`  <w:abstractNum w:abstractNumId="0">
    <w:lvl w:ilvl="0">
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="&#x2022;"/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="720"/>
      </w:pPr>
      <w:rPr>
        <w:rFonts w:ascii="Symbol" w:hAnsi="Symbol"/>
      </w:rPr>
    </w:lvl>
    <w:lvl w:ilvl="1">
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="o"/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="1440"/>
      </w:pPr>
      <w:rPr>
        <w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/>
      </w:rPr>
    </w:lvl>
    <w:lvl w:ilvl="2">
      <w:numFmt w:val="bullet"/>
      <w:lvlText w:val="&#x25AA;"/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="2160"/>
      </w:pPr>
      <w:rPr>
        <w:rFonts w:ascii="Wingdings" w:hAnsi="Wingdings"/>
      </w:rPr>
    </w:lvl>
  </w:abstractNum>
`)

	// Create one abstractNum for numbered lists
	buf.WriteString(`  <w:abstractNum w:abstractNumId="1">
    <w:lvl w:ilvl="0">
      <w:numFmt w:val="decimal"/>
      <w:lvlText w:val="%1."/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="720"/>
      </w:pPr>
    </w:lvl>
    <w:lvl w:ilvl="1">
      <w:numFmt w:val="lowerLetter"/>
      <w:lvlText w:val="%2."/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="1440"/>
      </w:pPr>
    </w:lvl>
    <w:lvl w:ilvl="2">
      <w:numFmt w:val="lowerRoman"/>
      <w:lvlText w:val="%3."/>
      <w:pPr>
        <w:widowControl w:val="1"/>
        <w:ind w:hanging="360" w:left="2160"/>
      </w:pPr>
    </w:lvl>
  </w:abstractNum>
`)

	// Generate num references for each list
	// Each list gets two num references: one for bullet (odd numId) and one for numbered (even numId)
	for i := range numLists {
		// Bullet list num (odd numId: 1, 3, 5, ...)
		buf.WriteString(fmt.Sprintf(`  <w:num w:numId="%d">
    <w:abstractNumId w:val="0"/>
  </w:num>
`, i*2+1))

		// Numbered list num (even numId: 2, 4, 6, ...)
		buf.WriteString(fmt.Sprintf(`  <w:num w:numId="%d">
    <w:abstractNumId w:val="1"/>
  </w:num>
`, i*2+2))
	}

	buf.WriteString(`</w:numbering>`)

	return buf.Bytes()
}

// generateSettingsXML creates word/settings.xml with basic document settings.
func generateSettingsXML() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:zoom w:percent="100"/>
  <w:defaultTabStop w:val="720"/>
  <w:characterSpacingControl w:val="doNotCompress"/>
  <w:compat>
    <w:useFELayout/>
  </w:compat>
</w:settings>`)
}

// injectTableStyles adds table-related styles to styles.xml content.
func injectTableStyles(stylesContent []byte) []byte {
	tableStyles := `
    <w:style w:type="table" w:styleId="TableGrid">
      <w:name w:val="Table Grid"/>
      <w:basedOn w:val="TableNormal"/>
      <w:uiPriority w:val="39"/>
      <w:tblPr>
        <w:tblBorders>
          <w:top w:val="single" w:sz="4" w:space="0" w:color="auto"/>
          <w:left w:val="single" w:sz="4" w:space="0" w:color="auto"/>
          <w:bottom w:val="single" w:sz="4" w:space="0" w:color="auto"/>
          <w:right w:val="single" w:sz="4" w:space="0" w:color="auto"/>
          <w:insideH w:val="single" w:sz="4" w:space="0" w:color="auto"/>
          <w:insideV w:val="single" w:sz="4" w:space="0" w:color="auto"/>
        </w:tblBorders>
      </w:tblPr>
    </w:style>
    <w:style w:type="table" w:default="1" w:styleId="TableNormal">
      <w:name w:val="Normal Table"/>
      <w:uiPriority w:val="99"/>
      <w:tblPr>
        <w:tblInd w:w="0" w:type="dxa"/>
        <w:tblCellMar>
          <w:top w:w="0" w:type="dxa"/>
          <w:left w:w="108" w:type="dxa"/>
          <w:bottom w:w="0" w:type="dxa"/>
          <w:right w:w="108" w:type="dxa"/>
        </w:tblCellMar>
      </w:tblPr>
    </w:style>
`

	// Find the closing </w:styles> tag and insert table styles before it
	closingTag := "</w:styles>"
	idx := bytes.Index(stylesContent, []byte(closingTag))
	if idx == -1 {
		return stylesContent
	}

	var result bytes.Buffer
	result.Write(stylesContent[:idx])
	result.WriteString(tableStyles)
	result.WriteString("\n")
	result.Write(stylesContent[idx:])

	return result.Bytes()
}

// injectHyperlinkStyles adds hyperlink styles to styles.xml content.
func injectHyperlinkStyles(stylesContent []byte) []byte {
	hyperlinkStyles := `
    <w:style w:type="character" w:styleId="Hyperlink">
      <w:name w:val="Hyperlink"/>
      <w:basedOn w:val="DefaultParagraphFont"/>
      <w:uiPriority w:val="99"/>
      <w:rPr>
        <w:color w:val="0563C1"/>
        <w:u w:val="single"/>
      </w:rPr>
    </w:style>
    <w:style w:type="character" w:styleId="FollowedHyperlink">
      <w:name w:val="Followed Hyperlink"/>
      <w:basedOn w:val="DefaultParagraphFont"/>
      <w:uiPriority w:val="99"/>
      <w:rPr>
        <w:color w:val="954F72"/>
        <w:u w:val="single"/>
      </w:rPr>
    </w:style>
`

	// Find the closing </w:styles> tag and insert hyperlink styles before it
	closingTag := "</w:styles>"
	idx := bytes.Index(stylesContent, []byte(closingTag))
	if idx == -1 {
		return stylesContent
	}

	var result bytes.Buffer
	result.Write(stylesContent[:idx])
	result.WriteString(hyperlinkStyles)
	result.WriteString("\n")
	result.Write(stylesContent[idx:])

	return result.Bytes()
}

// injectRelationships adds numbering, settings, and hyperlink relationships to document.xml.rels.
func injectRelationships(relsContent []byte, hyperlinks []HyperlinkInfo) []byte {
	// Find the closing </Relationships> tag
	closingTag := "</Relationships>"
	idx := bytes.Index(relsContent, []byte(closingTag))
	if idx == -1 {
		return relsContent
	}

	// Find the maximum existing rId to avoid conflicts
	maxRId := 0
	content := string(relsContent)
	for i := 1; i <= 20; i++ {
		if bytes.Contains(relsContent, []byte(fmt.Sprintf(`Id="rId%d"`, i))) {
			maxRId = i
		}
	}

	// Add new relationships
	var newRels strings.Builder
	newRels.WriteString(fmt.Sprintf(`
  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/numbering" Target="numbering.xml"/>
  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/settings" Target="settings.xml"/>
`, maxRId+1, maxRId+2))

	// Add hyperlink relationships
	for _, hyperlink := range hyperlinks {
		hyperlinkRel := fmt.Sprintf(
			`  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/hyperlink" Target="%s" TargetMode="External"/>
`, hyperlink.ID, hyperlink.URL)
		newRels.WriteString(hyperlinkRel)
	}

	var result bytes.Buffer
	result.Write(relsContent[:idx])
	result.WriteString(newRels.String())
	result.WriteString("\n")
	result.Write(relsContent[idx:])

	_ = content // suppress unused warning
	return result.Bytes()
}

// injectContentTypes adds numbering and settings content types to [Content_Types].xml.
func injectContentTypes(contentTypesData []byte) []byte {
	// Find the closing </Types> tag
	closingTag := "</Types>"
	idx := bytes.Index(contentTypesData, []byte(closingTag))
	if idx == -1 {
		return contentTypesData
	}

	// Add new content types
	newTypes := `
    <Override PartName="/word/numbering.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.numbering+xml"/>
    <Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>
`

	var result bytes.Buffer
	result.Write(contentTypesData[:idx])
	result.WriteString(newTypes)
	result.WriteString("\n")
	result.Write(contentTypesData[idx:])

	return result.Bytes()
}
