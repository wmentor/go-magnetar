package ipynb

import (
	"fmt"

	"github.com/wmentor/joveleaf"
)

type Codec struct{}

func (c *Codec) ReadFile(filename string) (string, error) {
	result, err := joveleaf.ReadToMarkdown(filename)
	if err != nil {
		return "", fmt.Errorf("read file %s error: %w", filename, err)
	}

	return result, nil
}
