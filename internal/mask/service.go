package mask

import (
	"context"
	"regexp"

	"github.com/wmentor/go-magnetar/internal/config"
)

const (
	defaultAlloc = 32
)

type record struct {
	name        string
	re          *regexp.Regexp
	replaceText string
}

type Service struct {
	records []record
}

func NewService(cfg *config.Config) *Service {
	srv := &Service{
		records: make([]record, 0, defaultAlloc),
	}

	if !cfg.Bool("mask.enable") {
		return srv
	}

	if !cfg.Bool("mask.skip.email") {
		srv.register("email", `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, "[MASKED_EMAIL]")
	}

	srv.register("github_token", `\b(?:gh[pousr]_[a-zA-Z0-9]{36,255}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})\b`, "[GITHUB_TOKEN]")
	srv.register("gitlab_token", `\bgl(?:pat|dt|ft|oas|rt|ptt|imt|cbt|agent)-[a-zA-Z0-9._-]{20,64}\b`, "[GITLAB_TOKEN]")

	return srv
}

func (srv *Service) Process(_ context.Context, input string) string {
	for _, rec := range srv.records {
		input = rec.re.ReplaceAllString(input, rec.replaceText)
	}
	return input
}

func (srv *Service) register(name string, pattern string, replaceText string) {
	srv.records = append(srv.records, record{
		name:        name,
		re:          regexp.MustCompile(pattern),
		replaceText: replaceText,
	})
}
