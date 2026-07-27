package docx

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fumiama/go-docx"
	"github.com/pkg/errors"
)

func ReadFile(filename string) (string, error) {
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
			if lvl, ok := getListLevel(item); ok {
				str := strings.ReplaceAll(strings.TrimLeft(fmt.Sprint(it), " \t"), "\n", " ")
				str = strings.Repeat("  ", lvl) + "* " + str
				fmt.Fprintln(&buffer, str)
			} else {
				fmt.Fprintln(&buffer, it)
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
