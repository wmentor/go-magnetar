package fetchplugin

import (
	"os/user"
	"path/filepath"
	"testing"
)

func TestOutputPath(t *testing.T) {
	t.Parallel()

	usr, err := user.Current()
	if err != nil {
		t.Skipf("cannot determine current user: %v", err)
	}
	home := usr.HomeDir

	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "tilde with path",
			arg:  "~/Downloads/article.md",
			want: filepath.Join(home, "Downloads", "article.md"),
		},
		{
			name: "tilde alone",
			arg:  "~",
			want: home,
		},
		{
			name: "tilde path is cleaned",
			arg:  "~/Downloads/../article.md",
			want: filepath.Join(home, "article.md"),
		},
		{
			name: "relative path",
			arg:  "./article.md",
			want: "article.md",
		},
		{
			name: "absolute path is cleaned",
			arg:  "/tmp/../tmp/article.md",
			want: "/tmp/article.md",
		},
		{
			name: "tilde in the middle is kept",
			arg:  "dir/~/article.md",
			want: "dir/~/article.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := outputPath(tt.arg); got != tt.want {
				t.Errorf("outputPath(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}
