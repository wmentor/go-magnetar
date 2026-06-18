package web

import (
	"fmt"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
)

func HTMLToMarkdown(html string) (string, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),             // Essential DOM/Node pruning behavior
			commonmark.NewCommonmarkPlugin(), // Implements standard CommonMark specs
			table.NewTablePlugin(),           // Enables GFM-compliant table rendering
		),
	)

	result, err := conv.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("convert html to markdown error: %w", err)
	}

	return result, nil
}
