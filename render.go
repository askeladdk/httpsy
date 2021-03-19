package httpsy

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
)

// Renderer serialises data to a writer.
type Renderer interface {
	Render(w io.Writer, header http.Header, data interface{}) error
}

// RendererFunc adapts a function to a Renderer.
type RendererFunc func(w io.Writer, header http.Header, data interface{}) error

// Render implements Renderer.
func (f RendererFunc) Render(w io.Writer, header http.Header, data interface{}) error {
	return f(w, header, data)
}

// JSONRenderer serialises data to a JSON object.
type JSONRenderer struct {
	Prefix, Indent string
	EscapeHTML     bool
}

// Render implements Renderer.
func (r JSONRenderer) Render(w io.Writer, header http.Header, data interface{}) error {
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")
	}
	e := json.NewEncoder(w)
	e.SetIndent(r.Prefix, r.Indent)
	e.SetEscapeHTML(r.EscapeHTML)
	return e.Encode(data)
}

// TemplateRenderer renders an HTML template.
type TemplateRenderer struct {
	Template *template.Template
	Name     string
}

// Render implements Renderer.
func (r TemplateRenderer) Render(w io.Writer, header http.Header, data interface{}) error {
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "text/html; charset=utf-8")
		header.Set("X-Content-Type-Options", "nosniff")
	}
	return r.Template.ExecuteTemplate(w, r.Name, data)
}

// Render writes the header and renders the data to the response.
// If the renderer returns an error, the response will be an HTTP 500 internal server error.
// The renderer is buffered so that no partial results become visible to the client.
func Render(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}, renderer Renderer) {
	buffer := getBytesBuffer()
	defer putBytesBuffer(buffer)

	if err := renderer.Render(buffer, w.Header(), data); err != nil {
		InternalServerError(w, r, err)
		return
	}

	w.WriteHeader(statusCode)
	_, _ = buffer.WriteTo(w)
}

// JSON is a convenience function that wraps JSONRenderer to reply with a JSON object.
func JSON(w http.ResponseWriter, r *http.Request, statusCode int, data interface{}) {
	Render(w, r, statusCode, data, JSONRenderer{})
}
