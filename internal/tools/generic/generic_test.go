package generic

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wmentor/go-magnetar/internal/config"
	"github.com/wmentor/go-magnetar/internal/plugin"
)

func newTestTools(t *testing.T, yaml string, readOnly bool) *GenericTools {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("version: \"1.0\"\n"+yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	return New(cfg, nil, &plugin.State{Config: cfg, ReadOnly: readOnly})
}

func TestSSHExecEnableFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		yaml     string
		readOnly bool
		want     string
	}{
		{
			name: "disabled by default",
			yaml: "",
			want: "error: ssh disabled",
		},
		{
			name: "explicitly disabled",
			yaml: "ssh:\n  enable: false\n",
			want: "error: ssh disabled",
		},
		{
			// With ssh enabled the call must get past the enable check.
			// An empty address stops it right after, before any network access.
			name: "enabled",
			yaml: "ssh:\n  enable: true\n",
			want: "error: no address provided",
		},
		{
			name:     "enabled but read-only mode",
			yaml:     "ssh:\n  enable: true\n",
			readOnly: true,
			want:     "error: the ssh tool is forbidden in read-only mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			g := newTestTools(t, tt.yaml, tt.readOnly)

			if got := g.SSHExec("ls", "", "", "", ""); got != tt.want {
				t.Errorf("SSHExec() = %q, want %q", got, tt.want)
			}
		})
	}
}
