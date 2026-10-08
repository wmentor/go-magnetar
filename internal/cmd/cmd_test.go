package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
)

// captureStdout runs fn and returns everything it printed to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	return string(out)
}

func TestPrintEnabledModules(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		want    []string
		notWant []string
	}{
		{
			name:    "nothing enabled",
			yaml:    "",
			notWant: []string{"ssh plugin is enabled", "guard plugin is enabled"},
		},
		{
			name:    "ssh enabled",
			yaml:    "ssh:\n  enable: true\n",
			want:    []string{"ssh plugin is enabled"},
			notWant: []string{"guard plugin is enabled"},
		},
		{
			name:    "guard enabled",
			yaml:    "guard:\n  enable: true\n",
			want:    []string{"guard plugin is enabled"},
			notWant: []string{"ssh plugin is enabled"},
		},
		{
			name: "ssh and guard enabled",
			yaml: "ssh:\n  enable: true\nguard:\n  enable: true\n",
			want: []string{"ssh plugin is enabled", "guard plugin is enabled"},
		},
		{
			name:    "ssh and guard explicitly disabled",
			yaml:    "ssh:\n  enable: false\nguard:\n  enable: false\n",
			notWant: []string{"ssh plugin is enabled", "guard plugin is enabled"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yml")
			if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}

			cfg, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}

			out := captureStdout(t, func() { printEnabledModules(cfg) })

			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("output %q does not contain %q", out, s)
				}
			}
			for _, s := range tt.notWant {
				if strings.Contains(out, s) {
					t.Errorf("output %q unexpectedly contains %q", out, s)
				}
			}
		})
	}
}
