package pages

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/wmentor/iwaread"
)

var ErrInvalidFileFormat = errors.New("invalid file format")

type Codec struct{}

func (c *Codec) ReadFile(filename string) (string, error) {
	content, err := iwaread.ReadPages(filename)
	if err != nil {
		return "", fmt.Errorf("read file %q error: %w", filename, err)
	}
	return content, nil
}
