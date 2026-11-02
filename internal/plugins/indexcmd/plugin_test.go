package indexcmd

import (
	"testing"
)

func TestParseIndexArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    string
		want    IndexArgs
		wantErr bool
	}{
		{
			name: "target only",
			args: "doc.md",
			want: IndexArgs{target: "doc.md"},
		},
		{
			name: "url target",
			args: "https://example.com/article",
			want: IndexArgs{target: "https://example.com/article"},
		},
		{
			name: "short recursive flag",
			args: "docs/ -r",
			want: IndexArgs{target: "docs/", recursive: true},
		},
		{
			name: "long recursive flag",
			args: "docs/ --recursive",
			want: IndexArgs{target: "docs/", recursive: true},
		},
		{
			name: "message flag",
			args: "doc.md -m context",
			want: IndexArgs{target: "doc.md", message: "context"},
		},
		{
			name: "recursive and message",
			args: "project/ -r -m context",
			want: IndexArgs{target: "project/", message: "context", recursive: true},
		},
		{
			name: "message before recursive",
			args: "project/ -m context -r",
			want: IndexArgs{target: "project/", message: "context", recursive: true},
		},
		{
			name: "flags before target",
			args: "-r -m context project/",
			want: IndexArgs{target: "project/", message: "context", recursive: true},
		},
		{
			name:    "empty input",
			args:    "",
			wantErr: true,
		},
		{
			name:    "flags without target",
			args:    "-r",
			wantErr: true,
		},
		{
			name:    "message flag without value",
			args:    "doc.md -m",
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    "doc.md -x",
			wantErr: true,
		},
		{
			name:    "uppercase flag is unknown",
			args:    "docs/ -R",
			wantErr: true,
		},
		{
			name:    "two targets",
			args:    "a.md b.md",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseIndexArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseIndexArgs(%q) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseIndexArgs(%q) = %+v, want %+v", tt.args, got, tt.want)
			}
		})
	}
}
