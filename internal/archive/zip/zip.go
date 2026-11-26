package zip

import (
	"archive/zip"
	"io"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/wmentor/go-magnetar/internal/archive"
)

func init() {
	// Register the ZIP archive handler
	archive.Register(&Archive{}, []string{".zip"}, []string{"application/zip"})
}

// Archive implements the archive.Archive interface for ZIP files.
type Archive struct{}

// List returns information about all files in the ZIP archive.
func (a *Archive) List(filename string) ([]archive.FileInfo, error) {
	r, err := zip.OpenReader(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open zip archive")
	}
	defer r.Close()

	files := make([]archive.FileInfo, 0, len(r.File))
	for _, f := range r.File {
		files = append(files, archive.FileInfo{
			Name:    f.Name,
			Size:    int64(f.UncompressedSize64),
			IsDir:   f.FileInfo().IsDir(),
			ModTime: f.Modified,
		})
	}

	return files, nil
}

// ReadFile reads the entire contents of a file within the ZIP archive.
func (a *Archive) ReadFile(archivePath, filePath string) ([]byte, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open zip archive")
	}
	defer r.Close()

	// Find the file (with case-insensitive fallback)
	f := findFile(r.File, filePath)
	if f == nil {
		return nil, archive.ErrFileNotFound
	}

	rc, err := f.Open()
	if err != nil {
		return nil, errors.Wrap(err, "failed to open file in archive")
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file from archive")
	}

	return data, nil
}

// findFile finds a file in the ZIP archive by path (case-insensitive fallback).
func findFile(files []*zip.File, target string) *zip.File {
	target = path.Clean(target)

	// First try exact match
	for _, f := range files {
		if path.Clean(f.Name) == target {
			return f
		}
	}

	// Case-insensitive fallback
	lowerTarget := strings.ToLower(target)
	for _, f := range files {
		if strings.ToLower(path.Clean(f.Name)) == lowerTarget {
			return f
		}
	}

	return nil
}
