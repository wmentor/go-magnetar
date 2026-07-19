package pdf

import (
	"bytes"

	"github.com/dslipak/pdf"
	"github.com/pkg/errors"
)

func ReadPDF(filename string) (string, error) {
	rf, err := pdf.Open(filename)
	if err != nil {
		return "", errors.Wrapf(err, "read pdf file %q error", filename)
	}

	pages := rf.NumPage()

	buf := bytes.NewBuffer(nil)

	for i := 1; i <= pages; i++ {
		page := rf.Page(i)
		rows, err := page.GetTextByRow()
		if err != nil {
			return "", errors.Wrap(err, "read page rows")
		}

		for _, row := range rows {
			for _, word := range row.Content {
				if buf.Len() > 0 {
					buf.WriteString(" ")
				}
				buf.WriteString(word.S)
			}
		}
	}

	return buf.String(), nil
}
