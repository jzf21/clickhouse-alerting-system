package api

import (
	"embed"
	"io/fs"
	"net/http"
	"path/filepath"
)

// UIFS is set from main.go with the embedded UI files.
var UIFS embed.FS

func serveSPA() http.Handler {
	sub, _ := fs.Sub(UIFS, "ui/dist")
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve a static file; fall back to index.html for SPA routing
		path := r.URL.Path
		if path == "/" {
			serveIndex(w, sub)
			return
		}
		ext := filepath.Ext(path)
		if ext == "" {
			// No extension means it's an SPA route, serve index.html
			serveIndex(w, sub)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
