package mask

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/wmentor/go-magnetar/internal/config"
)

const (
	defaultAlloc = 32
)

type record struct {
	name        string
	re          *regexp.Regexp
	replaceText string
	replaceFn   func(string) string
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
		// Must have separators between digit groups or be in format +XX XXX XXX XXX or similar
		phonePattern := `(?:\+\d{1,3}[-.\s]?)?(?:\(\d{3}\)|\b\d{3})[-.\s]\d{3}[-.\s](?:\d{4}|\d{2}[-.\s]\d{2})\b|` +
			`\b8[-.\s]?(?:\(\d{3}\)|\d{3})[-.\s]\d{3}[-.\s]\d{2}[-.\s]\d{2}\b|` +
			`\+\d{1,4}[-.\s]?(?:\(?\d{1,4}\)?[-.\s]?){2,4}\d{2,4}\b`
		srv.register("phone", phonePattern, "[MASKED_PHONE]")
	}

	if !cfg.Bool("mask.skip.iban") {
		srv.register("iban", `\b[A-Za-z]{2}\d{2}(?:\s[A-Za-z0-9]{4}){2,7}(?:\s[A-Za-z0-9]{1,4})?\b|\b[A-Za-z]{2}\d{2}[A-Za-z0-9]{11,30}\b`, "[MASK_IBAN]")
	}

	if !cfg.Bool("mask.skip.jwt") {
		srv.registerFunc("jwt_token", `\b[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+`, func(match string) string {
			if isJWT(match) {
				return "[MASK_TOKEN]"
			}
			return match
		})
	}

	if !cfg.Bool("mask.skip.ip") {
		rawRules := cfg.Strings("mask.skip.ip")
		if len(rawRules) == 0 {
			if s := cfg.String("mask.skip.ip"); s != "" && s != "false" {
				rawRules = []string{s}
			}
		}
		skipRules := parseIPSkipRules(rawRules)
		ipPattern := `(?:\b(?:\d{1,4}\.){3,}\d{1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,7}:[0-9a-fA-F]{1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}(?::[0-9a-fA-F]{1,4}){1,1}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,5}:[0-9a-fA-F]{1,4}(?::[0-9a-fA-F]{1,4}){1,2}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,4}:[0-9a-fA-F]{1,4}(?::[0-9a-fA-F]{1,4}){1,3}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,3}:[0-9a-fA-F]{1,4}(?::[0-9a-fA-F]{1,4}){1,4}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,2}:[0-9a-fA-F]{1,4}(?::[0-9a-fA-F]{1,4}){1,5}\b|` +
			`\b[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}\b|` +
			`\b:(?::[0-9a-fA-F]{1,4}){1,7}\b|` +
			`\b(?:[0-9a-fA-F]{1,4}:){1,7}:|` +
			`(?:\A|[^0-9a-zA-Z_:])(?:::1\b|::(?::[0-9a-fA-F]{1,4}){1,7}\b|::(?:\z|[^0-9a-zA-Z_:])))`
		srv.registerFunc("ip", ipPattern, func(match string) string {
			return maskIPAddress(match, skipRules)
		})
	}

	srv.register("github_token", `\b(?:gh[pousr]_[a-zA-Z0-9]{36,255}|github_pat_[a-zA-Z0-9]{22}_[a-zA-Z0-9]{59})\b`, "[MASKED_TOKEN]")
	srv.register("gitlab_token", `\bgl(?:pat|dt|ft|oas|rt|ptt|imt|cbt|agent)-[a-zA-Z0-9._-]{20,64}\b`, "[MASKED_TOKEN]")
	srv.register("aws_access_key", `\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`, "[MASKED_TOKEN]")
	srv.register("google_api_key", googleAPIKeyRe.String(), "[MASKED_TOKEN]")

	// Russian personal documents
	// Passport: 1234 567890 or 1234-567890 or 11 22 666666 (various space-separated formats)
	srv.register("passport", `\b\d{2}\s\d{2}\s\d{6}\b|\b\d{4}[\s-]\d{6}\b`, "[MASKED_PASSPORT]")
	// SNILS: 123-456-789 01 or 123-456-789-01 (3-3-3-2 format with dashes and optional space)
	srv.register("snils", `\b\d{3}-\d{3}-\d{3}[-\s]\d{2}\b`, "[MASKED_SNILS]")

	// US Social Security Number: XXX-XX-XXXX or XXX XX XXXX
	srv.register("ssn", `\b\d{3}[-\s]\d{2}[-\s]\d{4}\b`, "[MASK_SSN]")

	// Credit card number: XXXX XXXX XXXX XXXX or XXXX-XXXX-XXXX-XXXX (whitespace or dashes, not mixed)
	srv.register("card", `\b\d{4}(?:\s\d{4}){3}\b|\b\d{4}(?:-\d{4}){3}\b`, "[MASK_CARD]")

	return srv
}

func (srv *Service) Process(_ context.Context, input string) string {
	for _, rec := range srv.records {
		if rec.replaceFn != nil {
			input = rec.re.ReplaceAllStringFunc(input, rec.replaceFn)
		} else {
			input = rec.re.ReplaceAllString(input, rec.replaceText)
		}
	}
	return input
}

var googleAPIKeyRe = regexp.MustCompile(
	`(?s)\{[^{}]*"type"\s*:\s*"service_account"[^{}]*\}|` +
		`\b(?:AIza[0-9A-Za-z_-]{35}|ya29\.[0-9A-Za-z_-]{10,})\b`,
)

// MaskGoogleAPIKey masks Google Cloud API keys, OAuth 2.0 access tokens, and service account JSON keys.
func MaskGoogleAPIKey(input string) string {
	return googleAPIKeyRe.ReplaceAllString(input, "[MASKED_TOKEN]")
}

func (srv *Service) register(name string, pattern string, replaceText string) {
	srv.records = append(srv.records, record{
		name:        name,
		re:          regexp.MustCompile(pattern),
		replaceText: replaceText,
	})
}

func (srv *Service) registerFunc(name string, pattern string, fn func(string) string) {
	srv.records = append(srv.records, record{
		name:      name,
		re:        regexp.MustCompile(pattern),
		replaceFn: fn,
	})
}

func isJWT(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return false
	}

	headerBytes, err := decodeBase64URL(parts[0])
	if err != nil {
		return false
	}
	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return false
	}
	if _, ok := header["alg"]; !ok {
		return false
	}

	payloadBytes, err := decodeBase64URL(parts[1])
	if err != nil {
		return false
	}
	var payload any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return false
	}
	switch payload.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func decodeBase64URL(s string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}
