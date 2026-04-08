package main

import (
	"fmt"
	"html/template"
	"net/http"
)

type renderer struct {
	cache *template.Template
}

func newRenderer() *renderer {
	return &renderer{
		cache: template.Must(template.ParseGlob("ui/html/*.html")),
	}
}

func (r *renderer) render(w http.ResponseWriter, name string, data any) {
	if err := r.cache.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, fmt.Sprintf("Błąd w renderowaniu templatki: %v", err), http.StatusInternalServerError)
	}
}
