package html

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Garbage tags.
var junkTags = []string{
	"script", "style", "noscript", "iframe",
	"nav", "header", "footer", "aside",
	"button", "svg", "canvas", "input",
	"textarea",
	// "form",
}

// Garbage CSS-selectors (classes and IDs).
var junkSelectors = []string{
	"[class*='cookie']", "[class*='consent']",
	"[class*='subscribe']", "[class*='newsletter']",
	"[class*='social-share']", "[class*='share-buttons']",
	"[class*='related-posts']", "[class*='recommended']",
	"[class*='advertisement']", "[class*='ads-']",
	"[class*='ad-']", "[id*='ad-']",
	"[class*='popup']", "[class*='modal']",
	"[class*='widget']",
	"[class*='cta']", "[class*='call-to-action']",
	"[class*='tracking']", "[class*='analytics']",
	"[class*='paypal']", "[class*='bitcoin']",
	"[class*='donate']", "[class*='patreon']",
	"[class*='taboola']", "[class*='outbrain']",
	"[class*='disqus']", "[class*='fb-']",
	"[class*='twitter-']", "[class*='linkedin-']",
	"[id*='comments']", "[class*='comments']",
	"[class*='nav-']", "[id*='nav-']",
	"[class*='menu-']", "[id*='menu-']",
	"[role='navigation']", "[role='banner']", "[role='contentinfo']",
}

func cleanHTML(html string, page string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("parse HTML error: %w", err)
	}

	for _, tag := range junkTags {
		doc.Find(tag).Remove()
	}

	for _, sel := range junkSelectors {
		doc.Find(sel).Remove()
	}

	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		if strings.TrimSpace(s.Text()) == "" {
			s.Remove()
		}
	})

	var buf bytes.Buffer
	if html, err := doc.Html(); err == nil {
		buf.WriteString(html)
	}
	return buf.String(), nil
}
