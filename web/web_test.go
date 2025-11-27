package web

import (
	"io/fs"
	"testing"
)

func TestEmbeddedTemplatesExist(t *testing.T) {
	templatesFS := GetTemplatesFS()

	requiredFiles := []string{
		"admin/layout.html",
		"admin/dashboard.html",
		"admin/categories.html",
		"admin/results.html",
		"admin/voters.html",
		"admin/settings.html",
		"voter/vote.html",
	}

	for _, file := range requiredFiles {
		_, err := fs.Stat(templatesFS, file)
		if err != nil {
			t.Errorf("required template %q not found: %v", file, err)
		}
	}
}

func TestEmbeddedStaticFilesExist(t *testing.T) {
	staticFS := GetStaticFS()

	requiredFiles := []string{
		"css/admin.css",
		"js/admin.js",
		"js/dashboard.js",
		"js/categories.js",
		"js/results.js",
		"js/voters.js",
		"js/settings.js",
	}

	for _, file := range requiredFiles {
		_, err := fs.Stat(staticFS, file)
		if err != nil {
			t.Errorf("required static file %q not found: %v", file, err)
		}
	}
}

func TestTemplatesReadable(t *testing.T) {
	templatesFS := GetTemplatesFS()

	// Verify we can actually read content
	content, err := fs.ReadFile(templatesFS, "admin/layout.html")
	if err != nil {
		t.Fatalf("failed to read admin/layout.html: %v", err)
	}
	if len(content) == 0 {
		t.Error("admin/layout.html is empty")
	}
}

func TestStaticFilesReadable(t *testing.T) {
	staticFS := GetStaticFS()

	// Verify we can actually read content
	content, err := fs.ReadFile(staticFS, "js/admin.js")
	if err != nil {
		t.Fatalf("failed to read js/admin.js: %v", err)
	}
	if len(content) == 0 {
		t.Error("js/admin.js is empty")
	}
}
