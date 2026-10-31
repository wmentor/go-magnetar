package requester

import (
	"crypto/tls"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	maxTimeout = time.Hour * 2
)

type ClientFactory interface {
	Get() Client
	Put(c Client)
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// wrappedClient wraps http.Client to track request metrics.
type wrappedClient struct {
	client *http.Client
}

// Do implements Client interface and tracks request metrics.
func (w *wrappedClient) Do(req *http.Request) (*http.Response, error) {
	totalRequests.Add(1)

	// Determine request type based on URL path.
	path := req.URL.Path
	switch path {
	case "/v1/chat/completions":
		chatCompletionsRequests.Add(1)
	case "/v1/embeddings":
		embeddingRequests.Add(1)
	}

	//nolint:gosec // URL path is controlled internally
	resp, err := w.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		failedRequests.Add(1)
	}

	return resp, err
}

// Global counters for HTTP requests tracking.
var (
	totalRequests           atomic.Int64 // Total number of all requests
	chatCompletionsRequests atomic.Int64 // Number of v1/chat/completions requests
	embeddingRequests       atomic.Int64 // Number of v1/embeddings requests
	failedRequests          atomic.Int64 // Number of failed requests
)

var agent Client

func init() {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	baseClient := &http.Client{
		Transport: tr,
		Timeout:   maxTimeout,
	}

	agent = &wrappedClient{client: baseClient}
}

func Get() Client {
	return agent
}

// GetTotalRequests returns the total number of all HTTP requests.
func GetTotalRequests() int64 {
	return totalRequests.Load()
}

// GetChatCompletionsRequests returns the number of v1/chat/completions requests.
func GetChatCompletionsRequests() int64 {
	return chatCompletionsRequests.Load()
}

// GetEmbeddingRequests returns the number of v1/embeddings requests.
func GetEmbeddingRequests() int64 {
	return embeddingRequests.Load()
}

// GetFailedRequests returns the number of failed HTTP requests.
func GetFailedRequests() int64 {
	return failedRequests.Load()
}

func NewFactory() ClientFactory {
	return &factory{}
}

type factory struct{}

func (f *factory) Get() Client {
	return agent
}

func (f *factory) Put(_ Client) {
}
