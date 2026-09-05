// Package web embeds the built frontend and static assets into the binary.
package web

import "embed"

// FS holds static/ (the vite build output lives under static/dist) and
// robots.txt. The all: prefix makes the .gitkeep placeholder embeddable, so
// the Go build still succeeds before the frontend has been built.
//
//go:embed all:static robots.txt
var FS embed.FS
