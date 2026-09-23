package html

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
)

type Codec struct{}

type preprocessor func(content string, page string) (string, error)

var (
	preprocessors = []preprocessor{cleanHTML, processReadability, htmlToText}
)

func (c *Codec) ReadFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %w", filename, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filename, err)
	}

	fileURL, err := fileNameToURL(filename)
	if err != nil {
		return "", errors.Wrap(err, "convert filename to URL")
	}

	return c.ProcessContent(string(data), fileURL)
}

func (c *Codec) ProcessContent(content string, page string) (string, error) {
	for _, proc := range preprocessors {
		if ret, err := proc(content, page); err == nil {
			content = ret
		}
	}

	return content, nil
}
