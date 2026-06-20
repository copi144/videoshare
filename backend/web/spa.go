package web

import (
	_ "embed"
)

//go:embed spa/index.html
var spaHTML []byte

//go:embed spa/favicon.svg
var faviconSVG []byte

// SPA returns the embedded single-page application HTML file.
func SPA() []byte {
	return spaHTML
}

// Favicon returns the embedded favicon SVG file.
func Favicon() []byte {
	return faviconSVG
}
