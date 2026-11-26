package archive

import (
	"strings"
)

var (
	// archives maps file extensions to archive handlers.
	archives = make(map[string]Archive)

	// mimeArchives maps MIME types to archive handlers.
	mimeArchives = make(map[string]Archive)
)

// Register registers an archive handler for the given file extensions and MIME types.
// This is typically called from init() functions in archive implementation packages.
func Register(archive Archive, extensions []string, mimeTypes []string) {
	for _, ext := range extensions {
		archives[strings.ToLower(ext)] = archive
	}
	for _, mime := range mimeTypes {
		mimeArchives[strings.ToLower(mime)] = archive
	}
}

// GetArchive returns an archive handler for the given file extension.
func GetArchive(extension string) (Archive, error) {
	if a, ok := archives[strings.ToLower(extension)]; ok {
		return a, nil
	}
	return nil, ErrArchiveNotFound
}

// GetArchiveByMIME returns an archive handler for the given MIME type.
func GetArchiveByMIME(mime string) (Archive, error) {
	if a, ok := mimeArchives[strings.ToLower(mime)]; ok {
		return a, nil
	}
	return nil, ErrArchiveNotFound
}

// IsArchive checks if the given extension is a supported archive format.
func IsArchive(extension string) bool {
	_, ok := archives[strings.ToLower(extension)]
	return ok
}
