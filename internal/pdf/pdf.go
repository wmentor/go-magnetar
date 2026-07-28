package pdf

import (
	"github.com/pkg/errors"
	"github.com/razvandimescu/gopdf/pdf"
)

func ReadFile(filename string) (string, error) {
	doc, err := pdf.OpenFile(filename)
	if err != nil {
		return "", errors.Wrapf(err, "open file %s error", filename)
	}

	text, err := doc.Text()
	if err != nil {
		return "", errors.Wrapf(err, "get text from pdf file %s error", filename)
	}

	return text, nil
}
