package templates

import (
	"embed"
	"html/template"
	"io"
	"net/http"
)

//go:embed views/*.tmpl
var templateFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

// Templates holds the parsed templates
type Templates struct {
	templates *template.Template
}

// New creates a new Templates instance
func New() *Templates {
	funcMap := template.FuncMap{
		"json": jsonMarshal,
	}

	tmpl := template.Must(template.New("").Funcs(funcMap).ParseFS(templateFiles, "views/*.tmpl"))
	return &Templates{templates: tmpl}
}

// Render renders a template with the given data
func (t *Templates) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// StaticHandler returns an http.Handler for serving static files
func StaticHandler() http.Handler {
	return http.FileServerFS(staticFiles)
}
