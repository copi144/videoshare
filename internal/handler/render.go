package handler

import (
	"html/template"
	"io/fs"
	"net/http"
)

// TemplateData holds data injected into HTML templates.
type TemplateData struct {
	Title      string
	IsLoggedIn bool
	UserRole   string
	Username   string
	Resources  interface{}
	ResourceID string
	Error      string
	CSRFToken  string
	Data       interface{}
}

// parseAndRender parses and renders a named template with the base layout.
// data is passed as the template context (.).
func parseAndRender(w http.ResponseWriter, templates fs.FS, name string, data interface{}) error {
	tmpl, err := template.ParseFS(templates, "layout.html", name)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(w, name, data)
}
