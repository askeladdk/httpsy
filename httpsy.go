// Package httpsy (HTT-Peasy) extends and is compatible with the standard library
// http package to provide more tools for building HTTP services.
//
// Using the httpsy.ServeMux as a drop-in replacement for the http.ServeMux,
// it is easy to build Restful and other kinds of services. Httpsy uses the community standard
// middleware interface for maximum compatibility.
package httpsy

import (
	"context"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/askeladdk/httpsy/httpsyproblem"
)

// ContextKey maps values in a request context.
type ContextKey struct {
	name string
}

// NewContextKey allocates a context key that can be stored in a request context.
func NewContextKey(name string) *ContextKey {
	return &ContextKey{name}
}

var keyErrorHandler = NewContextKey("httpsy error handler")

// SetContextValue is a shorthand to map key to value in the request context.
func SetContextValue(r *http.Request, key *ContextKey, value interface{}) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, value))
}

// GetContextValue gets a value from the request context.
func GetContextValue(r *http.Request, key *ContextKey) interface{} {
	return r.Context().Value(key)
}

// ErrorHandler handles an error and generates an appropriate response.
type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, err error)
}

// ErrorHandlerFunc adapts a function to an ErrorHandler.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, err error)

// HandleError implements ErrorHandler.
func (f ErrorHandlerFunc) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	f(w, r, err)
}

// TextError replies to the request with the error in plain text.
func TextError(w http.ResponseWriter, r *http.Request, err error) {
	http.Error(w, err.Error(), StatusCode(err))
}

// JSONError replies to the request with the specified error.
func JSONError(w http.ResponseWriter, r *http.Request, err error) {
	// set content type if error is a ProblemDetailer
	if pd, ok := err.(httpsyproblem.Detailer); ok && pd.ProblemDetailer() {
		w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
	}

	JSON(w, r, StatusCode(err), err)
}

// Error replies to the request with the specified error message.
// It will use the error handler set with SetErrorHandler or defaults to TextError otherwise.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	var errorHandler ErrorHandler = ErrorHandlerFunc(TextError)
	if h, ok := GetContextValue(r, keyErrorHandler).(ErrorHandler); ok {
		errorHandler = h
	}
	errorHandler.HandleError(w, r, err)
}

// NotFound replies to the request with an HTTP 404 not found error.
func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, StatusNotFound)
}

// ShiftPath splits off the first component of p, which will be cleaned of
// relative components before processing. head will never contain a slash and
// tail will always be a rooted path without trailing slash.
func ShiftPath(p string) (head, tail string) {
	// taken from
	// https://blog.merovius.de/2017/06/18/how-not-to-use-an-http-router.html
	p = path.Clean("/" + p)
	if i := strings.Index(p[1:], "/") + 1; i > 0 {
		return p[1:i], p[i:]
	}
	return p[1:], "/"
}

// StripPrefix returns a handler that serves HTTP requests
// by removing the given prefix from the request URL's Path
// and invoking the handler h. StripPrefix handles a
// request for a path that doesn't begin with prefix by
// replying with an HTTP 404 not found error.
//
// This function is identical to http.StripPrefix with the exception
// that it calls httpsy.Error when the route does not match.
func StripPrefix(prefix string, h http.Handler) http.Handler {
	// taken from the Go source code but changed to hook
	// into the error handler by calling httpsy.Error.
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := strings.TrimPrefix(r.URL.Path, prefix); len(p) < len(r.URL.Path) {
			r2 := cloneRequestURL(r)
			r2.URL.Path = p
			h.ServeHTTP(w, r2)
			return
		}
		Error(w, r, StatusNotFound)
	})
}

// NoContent responds with HTTP 204 no content.
// Use it to implement healthcheck endpoints.
func NoContent(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

// Safe returns true if the request method is one of GET, HEAD, OPTIONS, TRACE.
// Safe methods can be cached and prefetched without any repercussions or side-effects to the resource.
func Safe(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

// NoListing disables directory listing in an http.FileSystem.
//
// How to use:
//  dir := httpsy.NoListing(http.Dir("."))
//  mux.Mount("/", http.FileServer(dir))
func NoListing(fs http.FileSystem) http.FileSystem {
	return noListing{fs}
}

type noListing struct {
	http.FileSystem
}

func (fs noListing) Open(name string) (http.File, error) {
	if f, err := fs.FileSystem.Open(name); err != nil {
		return nil, err
	} else if stat, err := f.Stat(); err != nil {
		_ = f.Close()
		return nil, err
	} else if stat.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	} else {
		return f, nil
	}
}
