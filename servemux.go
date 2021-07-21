package httpsy

import (
	"net/http"
	"strings"
)

// ServeMux is a drop-in replacement for http.ServeMux that understands middleware.
type ServeMux struct {
	serveMux    http.ServeMux
	middlewares Middlewares
}

// NewServeMux allocates and returns a new ServeMux.
func NewServeMux() *ServeMux {
	return new(ServeMux)
}

// Use appends one or more middlewares to the multiplexer.
func (mux *ServeMux) Use(mws ...MiddlewareFunc) {
	mux.middlewares = append(mux.middlewares, mws...)
}

// Handle adds a route and applies middlewares to it.
func (mux *ServeMux) Handle(pattern string, handler http.Handler) {
	mux.serveMux.Handle(pattern, mux.middlewares.Handler(Methods(handler)))
}

// HandleFunc adds a route and applies middlewares to it.
func (mux *ServeMux) HandleFunc(pattern string, handler http.HandlerFunc) {
	mux.Handle(pattern, handler)
}

// Mount attaches the handler along the pattern.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) Mount(pattern string, handler http.Handler) {
	if !strings.HasSuffix(pattern, "/") {
		panic("httpsy: pattern has no trailing slash")
	}
	handler = StripPrefix(strings.TrimSuffix(pattern, "/"), Methods(handler))
	mux.Handle(pattern, handler)
}

// MountFunc attaches the handler along the pattern.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) MountFunc(pattern string, handler http.HandlerFunc) {
	mux.Mount(pattern, handler)
}

// HandleParam mounts a route, applies middlewares to it,
// extracts the head URL parameter from the URL path,
// and stores it in the request's form values.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) HandleParam(pattern, param string, handler http.Handler) {
	mux.Mount(pattern, Param(param)(Methods(handler)))
}

// HandleParamFunc mounts a route, applies middlewares to it,
// extracts the head URL parameter from the URL path,
// and stores it in the request's form values.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) HandleParamFunc(pattern, param string, handler http.HandlerFunc) {
	mux.HandleParam(pattern, param, handler)
}

// Route mounts a new ServeMux along the pattern and applies the function to it.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) Route(pattern string, fn func(*ServeMux)) *ServeMux {
	newmux := NewServeMux()
	fn(newmux)
	mux.Mount(pattern, newmux)
	return newmux
}

// RouteParam mounts a new ServeMux along the pattern and applies the function to it.
//
// Panics if pattern does not have a trailing slash.
func (mux *ServeMux) RouteParam(pattern, param string, fn func(*ServeMux)) *ServeMux {
	newmux := NewServeMux()
	fn(newmux)
	mux.Mount(pattern, Param(param)(newmux))
	return newmux
}

// Handler returns the handler to use for the given request,
// consulting r.Method, r.Host, and r.URL.Path. It always returns
// a non-nil handler.
func (mux *ServeMux) Handler(r *http.Request) (h http.Handler, pattern string) {
	// hack to use httpsy error handling
	h, pattern = mux.serveMux.Handler(r)
	if pattern == "" {
		h = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Error(w, r, StatusNotFound)
		})
	}
	return
}

// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// hack to use httpsy error handling
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		Error(w, r, StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}
