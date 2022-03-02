package httpsy

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"sync"
)

// Renderer serialises a value to a writer.
type Renderer interface {
	Render(io.Writer, http.Header, interface{}) error
}

// JSONRenderer serialises data to a JSON object.
type JSONRenderer struct {
	Prefix, Indent string
	EscapeHTML     bool
}

// Render implements Renderer.
func (r JSONRenderer) Render(w io.Writer, h http.Header, d interface{}) error {
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set("X-Content-Type-Options", "nosniff")
	}
	e := json.NewEncoder(w)
	e.SetIndent(r.Prefix, r.Indent)
	e.SetEscapeHTML(r.EscapeHTML)
	return e.Encode(d)
}

// TemplateRenderer renders an HTML template.
type TemplateRenderer struct {
	Template *template.Template
	Name     string
}

// Render implements Renderer.
func (r TemplateRenderer) Render(w io.Writer, h http.Header, d interface{}) error {
	if h.Get("Content-Type") == "" {
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Set("X-Content-Type-Options", "nosniff")
	}
	return r.Template.ExecuteTemplate(w, r.Name, d)
}

var renderBufferPool = &sync.Pool{
	New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 4<<10)) },
}

// Render writes the header and renders the data to the response.
// If the renderer returns an error, the response will be an HTTP 500 internal server error.
// The renderer is buffered so that no partial results become visible to the client.
func Render(rr Renderer, w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	b := renderBufferPool.Get().(*bytes.Buffer)
	b.Reset()
	defer renderBufferPool.Put(b)

	if err := rr.Render(b, w.Header(), data); err != nil {
		Error(w, r, err)
		return
	}

	w.WriteHeader(code)
	_, _ = b.WriteTo(w)
}

// JSON is a convenience function that wraps JSONRenderer to reply with a JSON object.
func JSON(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	Render(JSONRenderer{EscapeHTML: true}, w, r, code, data)
}
