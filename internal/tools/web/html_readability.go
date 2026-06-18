package web

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	readability "codeberg.org/readeck/go-readability/v2"
)

func ProcessReadability(content, page string) (string, error) {
	baseURL, _ := url.Parse(page)

	article, err := readability.FromReader(strings.NewReader(content), baseURL)
	if err != nil {
		return "", fmt.Errorf("readability error: %w", err)
	}

	buf := bytes.NewBuffer(nil)

	if err = article.RenderHTML(buf); err != nil {
		return "", fmt.Errorf("readability render error: %w", err)
	}

	return buf.String(), nil
}
