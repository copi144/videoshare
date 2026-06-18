package web

import (
	"embed"
)

//go:embed spa/index.html
var spaFS embed.FS

// SPA returns the embedded single-page application HTML file.
func SPA() ([]byte, error) {
	return spaFS.ReadFile("spa/index.html")
}
