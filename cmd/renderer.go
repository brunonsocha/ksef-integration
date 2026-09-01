package main

import (
	"html/template"
	uiassets "ksef-integration/ui"
	"log"
	"net/http"
)

type renderer struct {
	cache    *template.Template
	errorLog *log.Logger
}

func newRenderer(errorLog *log.Logger) *renderer {
	return &renderer{
		cache:    template.Must(template.ParseFS(uiassets.Files, "html/*.html")),
		errorLog: errorLog,
	}
}

func (r *renderer) render(w http.ResponseWriter, name string, data any) {
	if err := r.cache.ExecuteTemplate(w, name, data); err != nil {
		r.errorLog.Printf("event=template_render_failed template=%q error=%q", name, err.Error())
		http.Error(w, "nie udało się wyświetlić strony.", http.StatusInternalServerError)
	}
}
