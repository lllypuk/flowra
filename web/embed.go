// Package web provides embedded static files and templates for the Flowra frontend.
package web

import "embed"

// TemplatesFS embeds all HTML templates from the templates directory.
// Use this for server-side rendering with html/template.
//
//go:embed templates
var TemplatesFS embed.FS

// StaticFS embeds all static assets from the static directory.
// Use this for serving CSS, JavaScript, and images.
//
//go:embed static
var StaticFS embed.FS
