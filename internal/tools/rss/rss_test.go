package rss_test

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/wmentor/go-magnetar/internal/tools/rss"
)

//go:embed testdata/rss.xml
var data []byte

func TestRSS(t *testing.T) {
	t.Parallel()

	feed, err := rss.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal("decode rss error")
	}

	if feed == nil {
		t.Fatal("feed empty")
	}

	if len(feed.Channel.Items) != 7 {
		t.Fatal("invalid items number")
	}

	result := feed.String()
	if len(result) != 1924 {
		t.Fatal("feed.String() invalid size")
	}
}
