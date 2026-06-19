package web

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Теги, которые всегда мусор
var junkTags = []string{
	"script", "style", "noscript", "iframe",
	"nav", "header", "footer", "aside",
	"button", "svg", "canvas", "input",
	"textarea",
	// "form",
}

// CSS-селекторы типичного мусора (классы и id)
var junkSelectors = []string{
	"[class*='cookie']", "[class*='consent']",
	"[class*='subscribe']", "[class*='newsletter']",
	"[class*='social-share']", "[class*='share-buttons']",
	"[class*='related-posts']", "[class*='recommended']",
	"[class*='advertisement']", "[class*='ads-']",
	"[id*='comments']", "[class*='comments']",
	"[role='navigation']", "[role='banner']", "[role='contentinfo']",
}

func CleanHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("парсинг HTML: %w", err)
	}

	// 1. Удаляем по тегам
	for _, tag := range junkTags {
		doc.Find(tag).Remove()
	}

	// 2. Удаляем по селекторам (классы/атрибуты)
	for _, sel := range junkSelectors {
		doc.Find(sel).Remove()
	}

	// 3. Удаляем пустые ссылки и якоря
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "" {
			s.Remove()
		}
	})

	// 4. Сериализуем обратно
	var buf bytes.Buffer
	if html, err := doc.Html(); err == nil {
		buf.WriteString(html)
	}
	return buf.String(), nil
}
