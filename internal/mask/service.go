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

	if !cfg.Bool("mask.skip.phone") {
		// Phone numbers: international (+X...), national formats (e.g. US (XXX) XXX-XXXX, Russian 8-XXX-XXX-XX-XX)
		srv.register("phone", `(?:\+\d{1,3}[-.\s]?)?(?:\(\d{3}\)|\b\d{3})[-.\s]?\d{3}[-.\s]?(?:\d{4}|\d{2}[-.\s]?\d{2})\b|\b8[-.\s]?(?:\(\d{3}\)|\d{3})[-.\s]?\d{3}[-.\s]?\d{2}[-.\s]?\d{2}\b|\+\d{1,4}[-.\s]?(?:\(?\d{1,4}\)?[-.\s]?){2,4}\d{2,4}\b`, "[MASKED_PHONE]")
	}

	srv.register("github_token", `\b(?:gh[pousr]_[a-zA-Z0-9]{36,255}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})\b`, "[MASKED_TOKEN]")
	srv.register("gitlab_token", `\bgl(?:pat|dt|ft|oas|rt|ptt|imt|cbt|agent)-[a-zA-Z0-9._-]{20,64}\b`, "[MASKED_TOKEN]")

	// Russian personal documents
	// Passport: 1234 567890 or 1234-567890 or 11 22 666666 (various space-separated formats)
	srv.register("passport", `\b\d{2}\s\d{2}\s\d{6}\b|\b\d{4}[\s-]\d{6}\b`, "[MASKED_PASSPORT]")
	// SNILS: 123-456-789 01 or 123-456-789-01 (3-3-3-2 format with dashes and optional space)
	srv.register("snils", `\b\d{3}-\d{3}-\d{3}[-\s]\d{2}\b`, "[MASKED_SNILS]")

	// US Social Security Number: XXX-XX-XXXX or XXX XX XXXX
	srv.register("ssn", `\b\d{3}[-\s]\d{2}[-\s]\d{4}\b`, "[MASK_SSN]")

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
