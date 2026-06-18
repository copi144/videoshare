package web

import (
	"embed"
	"io/fs"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

// When parsing these templates at runtime, use html/template (not text/template)
// for automatic context-aware XSS escaping. Example:
//   tmpl := template.Must(template.ParseFS(t, "*.html"))
func Templates() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic("failed to access embedded templates: " + err.Error())
	}
	return sub
}

func Static() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("failed to access embedded static files: " + err.Error())
	}
	return sub
}
