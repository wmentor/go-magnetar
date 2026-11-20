package plugin

import (
	"context"
	"errors"
	"net/http"
)

var ErrNoWebProcessor = errors.New("web processor skip URL")

type ClientFactory interface {
	Get() Client
	Put(c Client)
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type WebProcessorFunc func(ctx context.Context, url string) (string, error)

type WebProcessorStore interface {
	AddProcessorFunc(fn WebProcessorFunc)
	Process(ctx context.Context, url string) (string, error)
}
