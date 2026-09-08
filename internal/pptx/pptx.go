package pptx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

func getPageNumber(filename string) int {
	slideRegex := regexp.MustCompile(`ppt/slides/slide(\d+)\.xml`)
	matches := slideRegex.FindStringSubmatch(filename)

	if len(matches) < 2 {
		return -1
	}

	pageNum, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1
	}

	return pageNum
}

func ReadFile(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer r.Close()

	var sb strings.Builder

	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			pageNum := getPageNumber(f.Name)
			if pageNum == -1 {
				continue
			}

			rc, err := f.Open()
			if err != nil {
				return "", err
			}

			fmt.Fprintf(&sb, "\n\nPage %d:\n\n", pageNum)

			decoder := xml.NewDecoder(rc)
			for {
				token, err := decoder.Token()
				if err == io.EOF {
					break
				}
				if err != nil {
					rc.Close()
					return "", err
				}

				if startElement, ok := token.(xml.StartElement); ok && startElement.Name.Local == "t" {
					var content string
					if err := decoder.DecodeElement(&content, &startElement); err != nil {
						rc.Close()
						return "", err
					}
					sb.WriteString(content)
					sb.WriteString(" ")
				}
			}
			rc.Close()
		}
	}

	return sb.String(), nil
}
