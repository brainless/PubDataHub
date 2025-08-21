//go:build embed

package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var embeddedFiles embed.FS

// StaticHandler returns an http.Handler that serves embedded static files
func StaticHandler() (http.Handler, error) {
	distFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		return nil, err
	}
	return http.FileServer(http.FS(distFS)), nil
}
