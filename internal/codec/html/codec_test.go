package html_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wmentor/go-magnetar/internal/codec/html"
)

func TestCodec_ReadFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name: "simple HTML",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with script and styles",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title>
<style>body { color: black; }</style>
<script>console.log("test");</script>
</head>
<body>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with social share buttons",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="social-share">Share buttons</div>
<div class="share-buttons">Share this</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with newsletter subscription",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="newsletter">Subscribe to our newsletter</div>
<div class="subscribe">Subscribe</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with related posts",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="related-posts">Related articles</div>
<div class="recommended">You may also like</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with sidebar and widgets",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<aside class="sidebar">Sidebar content</aside>
<div class="widget">Widget</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with popup modal",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="modal">Modal popup</div>
<div class="popup">Popup</div>
<h1>Hello World</h1>
<p>This is a test.</div>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with comment sections",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div id="comments">Comments section</div>
<div class="comments">User comments</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with menu and navigation",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<header><nav class="nav-primary">Navigation</nav></header>
<div id="menu-main">Menu</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "empty HTML",
			html: `<!DOCTYPE html>
<html>
<head><title>Empty</title></head>
<body></body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with only text content",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
</body>
</html>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filename := filepath.Join(tmpDir, "test.html")

			if err := os.WriteFile(filename, []byte(tt.html), 0644); err != nil {
				t.Fatalf("write temp file error: %v", err)
			}

			codec := &html.Codec{}
			result, err := codec.ReadFile(filename)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ReadFile(%q) expected error, got nil", filename)
				}
				return
			}

			if err != nil {
				t.Fatalf("ReadFile(%q) unexpected error: %v", filename, err)
			}

			if result == "" && tt.name != "empty HTML" {
				t.Errorf("ReadFile(%q) returned empty string", filename)
			}

			if strings.Contains(result, "<script>") {
				t.Errorf("ReadFile(%q) did not remove script tags", filename)
			}

			if strings.Contains(result, "<style>") {
				t.Errorf("ReadFile(%q) did not remove style tags", filename)
			}

			if strings.Contains(result, "Cookie banner") {
				t.Errorf("ReadFile(%q) did not remove cookie banner", filename)
			}

			if strings.Contains(result, "Advertisement") && tt.name != "empty HTML" {
				t.Errorf("ReadFile(%q) did not remove ad container", filename)
			}
		})
	}
}

func TestCodec_ProcessContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		html    string
		wantErr bool
	}{
		{
			name: "simple HTML",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with navigation and footer",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<header>Header</header>
<nav>Navigation</nav>
<main>
<h1>Hello World</h1>
<p>This is a test.</p>
</main>
<footer>Footer</footer>
</body>
</html>`,
			wantErr: false,
		},
		{
			name: "HTML with various junk",
			html: `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body>
<div class="cookie-consent">Cookie</div>
<div class="ad-banner">Ad</div>
<div id="comments">Comments</div>
<div class="related-posts">Related</div>
<div class="newsletter">Newsletter</div>
<h1>Hello World</h1>
<p>This is a test.</p>
</body>
</html>`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			codec := &html.Codec{}
			result, err := codec.ProcessContent(tt.html, "http://example.com")

			if tt.wantErr {
				if err == nil {
					t.Errorf("ProcessContent() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ProcessContent() unexpected error: %v", err)
			}

			if result == "" && tt.name != "simple HTML" {
				t.Errorf("ProcessContent() returned empty string")
			}
		})
	}
}
