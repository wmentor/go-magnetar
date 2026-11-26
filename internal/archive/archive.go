package archive

import (
	"time"
)

// FileInfo represents metadata about a file within an archive.
type FileInfo struct {
	Name    string    `json:"name"`     // File path within the archive
	Size    int64     `json:"size"`     // File size in bytes
	IsDir   bool      `json:"is_dir"`   // Whether the entry is a directory
	ModTime time.Time `json:"mod_time"` // Last modification time
}

// Archive defines operations for reading archive contents.
type Archive interface {
	// List returns information about all files in the archive.
	List(filename string) ([]FileInfo, error)

	// ReadFile reads the entire contents of a file within the archive.
	ReadFile(archivePath, filePath string) ([]byte, error)
}
