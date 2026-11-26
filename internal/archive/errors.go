package archive

import "errors"

var (
	// ErrArchiveNotFound is returned when no archive handler is found for the given extension.
	ErrArchiveNotFound = errors.New("archive handler not found")

	// ErrFileNotFound is returned when the specified file is not found in the archive.
	ErrFileNotFound = errors.New("file not found in archive")
)
