package html

import (
	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/wmentor/html"
)

func htmlToText(content string, page string) (string, error) {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),             // Essential DOM/Node pruning behavior
			commonmark.NewCommonmarkPlugin(), // Implements standard CommonMark specs
			table.NewTablePlugin(),           // Enables GFM-compliant table rendering
		),
	)

	result, err := conv.ConvertString(content)
	if err != nil {
		parser := html.New()

		parser.ParseString(content)

		data := parser.Text()
		return string(data), nil
	}

	return result, nil
}
