package archive

import (
	"testing"
)

func TestGetArchive_NoHandlers(t *testing.T) {
	// Note: Not using t.Parallel() because this test modifies global state

	// Clear any registered handlers for this test
	originalArchives := archives
	archives = make(map[string]Archive)
	defer func() { archives = originalArchives }()

	_, err := GetArchive(".zip")
	if err != ErrArchiveNotFound {
		t.Errorf("GetArchive() error = %v, want %v", err, ErrArchiveNotFound)
	}
}

func TestGetArchiveByMIME_NoHandlers(t *testing.T) {
	// Note: Not using t.Parallel() because this test modifies global state

	// Clear any registered handlers for this test
	originalMimeArchives := mimeArchives
	mimeArchives = make(map[string]Archive)
	defer func() { mimeArchives = originalMimeArchives }()

	_, err := GetArchiveByMIME("application/zip")
	if err != ErrArchiveNotFound {
		t.Errorf("GetArchiveByMIME() error = %v, want %v", err, ErrArchiveNotFound)
	}
}

func TestIsArchive_NoHandlers(t *testing.T) {
	// Note: Not using t.Parallel() because this test modifies global state

	// Clear any registered handlers for this test
	originalArchives := archives
	archives = make(map[string]Archive)
	defer func() { archives = originalArchives }()

	if IsArchive(".zip") {
		t.Error("IsArchive() = true, want false when no handlers registered")
	}
}

func TestRegister(t *testing.T) {
	// Note: Not using t.Parallel() because this test modifies global state

	// Clear any registered handlers for this test
	originalArchives := archives
	originalMimeArchives := mimeArchives
	archives = make(map[string]Archive)
	mimeArchives = make(map[string]Archive)
	defer func() {
		archives = originalArchives
		mimeArchives = originalMimeArchives
	}()

	// Create a mock archive
	mockArchive := &mockArchiveImpl{}

	// Register it
	Register(mockArchive, []string{".test", ".TEST"}, []string{"application/test"})

	// Verify it's registered
	if _, err := GetArchive(".test"); err != nil {
		t.Errorf("GetArchive(.test) error = %v, want nil", err)
	}

	if _, err := GetArchive(".TEST"); err != nil {
		t.Errorf("GetArchive(.TEST) error = %v, want nil", err)
	}

	if _, err := GetArchiveByMIME("application/test"); err != nil {
		t.Errorf("GetArchiveByMIME(application/test) error = %v, want nil", err)
	}

	if !IsArchive(".test") {
		t.Error("IsArchive(.test) = false, want true")
	}
}

// mockArchiveImpl is a mock implementation for testing
type mockArchiveImpl struct{}

func (m *mockArchiveImpl) List(filename string) ([]FileInfo, error) {
	return nil, nil
}

func (m *mockArchiveImpl) ReadFile(archivePath, filePath string) ([]byte, error) {
	return nil, nil
}
