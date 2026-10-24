package requester

import (
	"crypto/tls"
	"net/http"
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

var agent *http.Client

func init() {
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
	}

	agent = &http.Client{
		Transport: tr,
		Timeout:   maxTimeout,
	}
}

func Get() Client {
	return agent
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
