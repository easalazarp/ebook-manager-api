package handlers

import (
	"html/template"
	"net/http"
	"path/filepath"
)

func render(w http.ResponseWriter, name string, data any) {
	tmpl, err := template.ParseFiles(
		filepath.Join("internal", "views", "layout.html"),
		filepath.Join("internal", "views", name),
	)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err = tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
