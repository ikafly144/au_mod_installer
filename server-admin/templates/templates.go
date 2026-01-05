package templates

import (
	"embed"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
)

//go:embed views/*.go.tmpl
var templateFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

// MIME types for static files
var mimeTypes = map[string]string{
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".html":  "text/html; charset=utf-8",
	".svg":   "image/svg+xml",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
}

// Templates holds the parsed templates
type Templates struct {
	base *template.Template
}

// New creates a new Templates instance
func New() *Templates {
	funcMap := template.FuncMap{
		"json": jsonMarshal,
	}

	// Load base layout template
	tmpl := template.New("base").Funcs(funcMap)
	tmpl = template.Must(tmpl.ParseFS(templateFiles, "views/layout.go.tmpl"))

	return &Templates{base: tmpl}
}

// Render renders a template with the given data
func (t *Templates) Render(w io.Writer, name string, data any) error {
	// Clone the base template
	tmpl, err := t.base.Clone()
	if err != nil {
		return err
	}

	// Parse the specific view template
	// Assuming the view file is named "views/{name}.go.tmpl"
	pattern := "views/" + name + ".go.tmpl"
	if _, err := tmpl.ParseFS(templateFiles, pattern); err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, name, data)
}

// staticHandler wraps http.Handler to set proper Content-Type headers
type staticHandler struct {
	handler http.Handler
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ext := filepath.Ext(r.URL.Path)
	if mimeType, ok := mimeTypes[ext]; ok {
		w.Header().Set("Content-Type", mimeType)
	}
	h.handler.ServeHTTP(w, r)
}

// StaticHandler returns an http.Handler for serving static files
func StaticHandler() http.Handler {
	return &staticHandler{handler: http.FileServerFS(staticFiles)}
}
