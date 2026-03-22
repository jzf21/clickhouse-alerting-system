package api

import (
	"embed"
	"net/http"
)

// UIFS is set from main.go with the embedded UI files.
var UIFS embed.FS

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := UIFS.ReadFile("ui/index.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func serveAppJS(w http.ResponseWriter, r *http.Request) {
	data, err := UIFS.ReadFile("ui/app.js")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(data)
}

func serveStyleCSS(w http.ResponseWriter, r *http.Request) {
	data, err := UIFS.ReadFile("ui/style.css")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css")
	w.Write(data)
}
