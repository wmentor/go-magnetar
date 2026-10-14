package web

import (
	"mime"
	"strings"
)

// GetMimeType extracts the MIME type from a content type string.
// It parses the content type and returns the media type without parameters (e.g., charset).
func GetMimeType(contentType string) (string, error) {
	if contentType == "" {
		return "", nil
	}

	// Parse and return the media type (without parameters like charset)
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}
	return strings.ToLower(mediaType), nil
}
