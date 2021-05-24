package httpsy

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/askeladdk/httpsy/httpsyproblem"
)

// MiddlewareFunc defines middleware.
type MiddlewareFunc func(http.Handler) http.Handler

// Middlewares is a slice of middlewares that are applied sequentially to an endpoint.
type Middlewares []MiddlewareFunc

// Chain creates a Middlewares type from its arguments.
func Chain(mws ...MiddlewareFunc) Middlewares {
	return Middlewares(mws)
}

// Handler applies the Middlewares to the endpoint.
func (mw Middlewares) Handler(endpoint http.Handler) http.Handler {
	if len(mw) == 0 {
		return endpoint
	}

	h := mw[len(mw)-1](endpoint)
	for i := len(mw) - 2; i >= 0; i-- {
		h = mw[i](h)
	}

	return h
}

// Endpoint adapts a handler into a middleware.
// This middleware does not nothing but return the handler.
// The handler then becomes the terminal endpoint of the middleware chain.
func Endpoint(h http.Handler) MiddlewareFunc {
	return func(_ http.Handler) http.Handler {
		return h
	}
}

// EndpointFunc adapts a handler function into a middleware.
// This middleware does not nothing but return the handler.
// The handler then becomes the terminal endpoint of the middleware chain.
func EndpointFunc(h http.HandlerFunc) MiddlewareFunc {
	return Endpoint(h)
}

// AcceptContentTypes only accepts requests that have the Content-Type headers
// set to one of the given content types.
// Other requests are responded to with an HTTP 415 unsupported media type.
// Content types are matched using path.Match and can contain wildcards.
func AcceptContentTypes(contentTypes ...string) MiddlewareFunc {
	cts := make([]string, len(contentTypes))
	for i, ct := range contentTypes {
		cts[i] = strings.ToLower(ct)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ct := r.Header.Get("Content-Type")
			if i := strings.Index(ct, ";"); i >= 0 {
				ct = ct[:i]
			}

			ct = strings.TrimSpace(ct)
			ct = strings.ToLower(ct)

			if !stringsMatch(cts, ct) {
				Error(w, r, StatusUnsupportedMediaType)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RealIP is a middleware that adjusts the request RemoteAddr field according
// to the IP address found in the X-Real-IP and X-Forwarded-For request headers
// if either exist. The port number in RemoteAddr is preserved.
//
// Only use this middleware when the server is behind a reverse proxy that sets
// these headers or you will make it too easy for attackers to spoof their IP address.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if xrip := r.Header.Get("X-Real-Ip"); xrip != "" {
			_, port, _ := net.SplitHostPort(r.RemoteAddr)
			r.RemoteAddr = net.JoinHostPort(xrip, port)
		} else if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			if i := strings.Index(xff, ","); i >= 0 {
				xff = xff[:i]
			}
			_, port, _ := net.SplitHostPort(r.RemoteAddr)
			r.RemoteAddr = net.JoinHostPort(xff, port)
		}
		next.ServeHTTP(w, r)
	})
}

// Authenticate is a middleware that delegates authentication to a function.
//
// The permit parameter authenticates a request, returning the (possibly modified) request
// and an error value denoting the success or failure of authentication.
// An error value of nil means that authentication succeeded.
// Permit should return StatusUnauthorized if authentication failed,
// and StatusForbidden if authentication succeeded but
// the user is still not allowed to access the resource.
// The request must always be returned along with the error.
// It may be modified using WithContextValue to set context values that
// endpoints or other middleware down the line can use.
func Authenticate(permit func(*http.Request) (*http.Request, error)) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r, err := permit(r)
			if err != nil {
				if StatusCode(err) == http.StatusUnauthorized {
					if w.Header().Get("WWW-Authenticate") == "" {
						w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="utf-8"`, r.Host))
					}
				}
				Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Param is a middleware that extracts the head URL parameter
// from the URL path and stores it as a form value.
// It is rarely needed to use Param on its own because
// it is used by ServeMux to implement the *Param methods.
//
//  Param("orderID") // create the middleware
//  r.FormValue("orderID") // get the parameter in the handler
//
// The URL parameter may optionally be given a regex constraint
// by adding a colon followed by the regular expression:
//  Param("orderID:[0-9]+$") // only matches integers
//
// It is also possible to use an empty name. In this case the regex
// constraint is applied but the value is not stored in the form values:
//  Param(":v[12]$") // routes /v1 and /v2 to the same handler
//
// However, an empty name without a regular expression is not allowed:
//  Param("") // panics
//  Param(":") // panics
func Param(param string) MiddlewareFunc {
	if param == "" || param == ":" {
		panic("httpsy: param cannot be empty string")
	}

	var rxp *regexp.Regexp
	name := param

	if i := strings.Index(param, ":"); i >= 0 {
		name, rxp = param[:i], regexp.MustCompile(param[i+1:])
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var head string
			r = cloneRequestURL(r)
			if head, r.URL.Path = ShiftPath(r.URL.Path); head == "" {
				Error(w, r, StatusNotFound)
				return
			} else if rxp != nil && !rxp.MatchString(head) {
				Error(w, r, StatusNotFound)
				return
			} else if err := r.ParseForm(); err != nil {
				Error(w, r, httpsyproblem.Wrap(err, http.StatusInternalServerError))
				return
			} else if name != "" {
				r.Form.Add(name, head)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// NoCache is a middleware that sets a number of HTTP headers to prevent
// a router (or subrouter) from being cached by an upstream proxy and/or client.
func NoCache(next http.Handler) http.Handler {
	// taken from https://github.com/go-chi/chi
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("ETag")
		r.Header.Del("If-Modified-Since")
		r.Header.Del("If-Match")
		r.Header.Del("If-None-Match")
		r.Header.Del("If-Range")
		r.Header.Del("If-Unmodified-Since")
		w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 UTC")
		w.Header().Set("Cache-Control", "no-cache, no-store, no-transform, must-revalidate, private, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Accel-Expires", "0")
		next.ServeHTTP(w, r)
	})
}

// WithContextValue maps the key to the value in the request context.
func WithContextValue(key *ContextKey, value interface{}) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, SetContextValue(r, key, value))
		})
	}
}

// SetErrorHandler is a middleware that sets the error handler used by Error.
func SetErrorHandler(errorHandler ErrorHandlerFunc) MiddlewareFunc {
	return WithContextValue(keyErrorHandler, errorHandler)
}

// Recoverer recovers from panics by responding with an HTTP 500 internal server error.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil && v != http.ErrAbortHandler {
				var err error
				switch x := v.(type) {
				case error:
					err = httpsyproblem.Wrap(x, http.StatusInternalServerError)
				case string:
					err = httpsyproblem.Wrap(errorString(x), http.StatusInternalServerError)
				default:
					err = httpsyproblem.Wrap(errorString(fmt.Sprintf("%v", v)), http.StatusInternalServerError)
				}
				Error(w, r, err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID generates a random X-Request-ID header if is it not already set.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Request-ID") == "" {
			var id [32]byte
			randomNoise(id[:])
			r.Header.Set("X-Request-ID", bytesToASCII(id[:]))
		}
		next.ServeHTTP(w, r)
	})
}

// Predicator accepts or rejects a request.
type Predicator interface {
	Predicate(*http.Request) bool
}

// PredicateFunc accepts or rejects a request.
type PredicateFunc func(*http.Request) bool

// Predicate implements Predicator.
func (f PredicateFunc) Predicate(r *http.Request) bool {
	return f(r)
}

// If applies the middleware only if the condition is true.
func If(cond Predicator, then MiddlewareFunc) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cond.Predicate(r) {
				next = then(next)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IfChain applies the middlewares only if the condition is true.
func IfChain(cond Predicator, then ...MiddlewareFunc) MiddlewareFunc {
	return If(cond, Chain(then...).Handler)
}
