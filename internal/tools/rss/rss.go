package rss

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/html/charset"

	"github.com/wmentor/go-magnetar/internal/codec/html"
)

type RSS struct {
	Channel Channel `xml:"channel"`
}

func (r *RSS) String() string {
	var sb strings.Builder

	codec := &html.Codec{}

	sb.WriteString("RSS Feed\n")

	for _, item := range r.Channel.Items {
		sb.WriteString("\n---\n\n")

		descr, err := codec.ProcessContent(item.Description, item.Link)
		if err != nil {
			fmt.Fprintf(&sb, "URL: %s\nTITLE: %s\nDESCRIPTION: %s\n", item.Link, item.Title, item.Description)
		} else {
			fmt.Fprintf(&sb, "URL: %s\nTITLE: %s\nDESCRIPTION: %s\n", item.Link, item.Title, descr)
		}
	}

	return sb.String()
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

func Decode(in io.Reader) (*RSS, error) {
	decoder := xml.NewDecoder(in)

	decoder.CharsetReader = charset.NewReaderLabel

	rss := new(RSS)

	if err := decoder.Decode(rss); err != nil {
		return nil, errors.Wrap(err, "decode rss xml")
	}

	return rss, nil
}
