package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// GetTemplatesFS returns the embedded templates filesystem
func GetTemplatesFS() fs.FS {
	sub, _ := fs.Sub(templatesFS, "templates")
	return sub
}

// GetStaticFS returns the embedded static files filesystem
func GetStaticFS() fs.FS {
	sub, _ := fs.Sub(staticFS, "static")
	return sub
}
