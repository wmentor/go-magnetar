package uuid

import (
	"bytes"
	"errors"
	"testing"

	"github.com/wmentor/go-magnetar/internal/http/requester"
	"github.com/wmentor/go-magnetar/internal/printer"
	"github.com/wmentor/go-magnetar/internal/tools"
)

var errRandomSource = errors.New("random source failed")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errRandomSource }

func TestGenerateUUID(t *testing.T) {
	t.Parallel()
	got, err := generateUUID(bytes.NewReader(bytes.Repeat([]byte{0x42}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	if got != "42424242-4242-4242-8242-424242424242" {
		t.Fatalf("generateUUID() = %q", got)
	}
}

func TestGenerateUUIDReaderFailure(t *testing.T) {
	t.Parallel()
	got, err := generateUUID(failingReader{})
	if !errors.Is(err, errRandomSource) {
		t.Fatalf("generateUUID() error = %v", err)
	}
	if got != "" {
		t.Fatalf("generateUUID() = %q, want empty string", got)
	}
}

func TestExec(t *testing.T) {
	t.Parallel()
	tool := New(nil, printer.GetDefault(), requester.NewFactory(), nil, nil)
	tests := []struct {
		name   string
		params *SystemUUIDParams
	}{
		{name: "nil params"},
		{name: "empty params", params: &SystemUUIDParams{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Exec(t.Context(), tt.params)
			if err != nil {
				t.Fatal(err)
			}
			response, ok := result.(*tools.Response)
			if !ok {
				t.Fatalf("Exec() result type = %T", result)
			}
			if response.Status != tools.StatusOK || response.Result == "" {
				t.Fatalf("Exec() response = %+v", response)
			}
		})
	}
}
