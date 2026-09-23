package html

import (
	"net/url"
	"path/filepath"

	"github.com/pkg/errors"
)

func fileNameToURL(filename string) (string, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", errors.Wrap(err, "calculate absolute file path")
	}

	u := &url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absPath),
	}

	fileURL := u.String()

	return fileURL, nil
}
