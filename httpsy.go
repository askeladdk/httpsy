// Package httpsy (HTT-Peasy) extends and is compatible with the standard library
// http package to provide more tools for building HTTP services.
//
// Using the httpsy.ServeMux as a drop-in replacement for the http.ServeMux,
// it is easy to build Restful and other kinds of services. Httpsy uses the community standard
// middleware interface for maximum compatibility.
package httpsy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
)

// ErrAuthenticationFailed is returned by an Authenticator to signal that authentication failed.
const ErrAuthenticationFailed = errorString("authentication failed")

// ErrAccessForbidden is returned by an Authenticator to signal that access is forbidden
// even though authentication was successful.
const ErrAccessForbidden = errorString("access forbidden")

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

// StatusError returns an error for the HTTP status code.
func StatusError(statusCode int) error {
	return errorString(http.StatusText(statusCode))
}

// ErrorHandler handles an error and generates an appropriate response.
type ErrorHandler interface {
	HandleError(w http.ResponseWriter, r *http.Request, statusCode int, err error)
}

// ErrorHandlerFunc adapts a function to an ErrorHandler.
type ErrorHandlerFunc func(w http.ResponseWriter, r *http.Request, statusCode int, err error)

// HandleError implements ErrorHandler.
func (f ErrorHandlerFunc) HandleError(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	f(w, r, statusCode, err)
}

// TextError responds with the status code and error string in plain text.
func TextError(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	http.Error(w, err.Error(), statusCode)
}

// JSONError replies to the request with the specified error message and HTTP code.
func JSONError(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	response := struct {
		Message string `json:"message"`
		Status  int    `json:"status"`
	}{err.Error(), statusCode}
	JSON(w, r, statusCode, response)
}

// Error replies to the request with the specified error message and HTTP code.
// It will use the error handler set with SetErrorHandler, otherwise it defaults to TextError.
func Error(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	var errorHandler ErrorHandler = ErrorHandlerFunc(TextError)
	if h, ok := GetContextValue(r, keyErrorHandler).(ErrorHandler); ok {
		errorHandler = h
	}
	errorHandler.HandleError(w, r, statusCode, err)
}

// BadRequest replies to the request with an HTTP 400 bad request.
func BadRequest(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusBadRequest, StatusError(http.StatusBadRequest))
}

// NotFound replies to the request with an HTTP 404 not found error.
func NotFound(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusNotFound, StatusError(http.StatusNotFound))
}

// MethodNotAllowed replies to the request with an HTTP 405 method not allowed error.
func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusMethodNotAllowed, StatusError(http.StatusMethodNotAllowed))
}

// Unauthorized replies to the request with an HTTP 401 unauthorized.
// It will set the WWW-Authenticate header to a default value if it was not already set.
func Unauthorized(w http.ResponseWriter, r *http.Request) {
	if w.Header().Get("WWW-Authenticate") == "" {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="utf-8"`, r.Host))
	}
	Error(w, r, http.StatusUnauthorized, StatusError(http.StatusUnauthorized))
}

// Forbidden replies to the request with an HTTP 403 forbidden.
func Forbidden(w http.ResponseWriter, r *http.Request) {
	Error(w, r, http.StatusForbidden, StatusError(http.StatusForbidden))
}

// InternalServerError replies to the request with an HTTP 500 internal server error.
func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	Error(w, r, http.StatusInternalServerError, err)
}

// Authenticator permits a request, returning the (possibly modified) request
// and an error value denoting the success or failure of authentication.
//
// An error value of nil means that authentication succeeded.
// A value of ErrAuthenticationFailed means that authentication failed.
// A value of ErrAccessForbidden means that authentication succeeded but
// the user is still not allowed to access the resource.
// Any other value means that an error occurred.
//
// The request must always be returned along with the error.
// It may be modified using WithContextValue to set context values that
// endpoints or other middleware down the line can use.
type Authenticator interface {
	Authenticate(*http.Request) (*http.Request, error)
}

// AuthenticatorFunc adapts a function to an Authenticator.
type AuthenticatorFunc func(*http.Request) (*http.Request, error)

// Authenticate implements Authenticator.
func (f AuthenticatorFunc) Authenticate(r *http.Request) (*http.Request, error) {
	return f(r)
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
	// into the error handler by calling httpsy.NotFound.
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
		NotFound(w, r)
	})
}

// Pong responds with HTTP 204 no content.
// Use it to implement healthcheck endpoints.
func Pong(w http.ResponseWriter, r *http.Request) {
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
