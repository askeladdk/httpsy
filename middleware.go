package httpsy

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/askeladdk/httpsyproblem"
)

// AcceptContentTypes only accepts requests that have the Content-Type headers
// set to one of the given content types.
// Other requests are responded to with an HTTP 415 unsupported media type.
func AllowContentType(contentTypes ...string) func(http.Handler) http.Handler {
	allowedContentTypes := make(map[string]struct{}, len(contentTypes))
	for _, ctype := range contentTypes {
		allowedContentTypes[strings.TrimSpace(strings.ToLower(ctype))] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength == 0 {
				// skip check for empty content body
				next.ServeHTTP(w, r)
				return
			}

			s := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
			if i := strings.Index(s, ";"); i > -1 {
				s = s[0:i]
			}

			if _, ok := allowedContentTypes[s]; ok {
				next.ServeHTTP(w, r)
				return
			}

			Error(w, r, httpsyproblem.StatusUnsupportedMediaType)
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

// BasicAuth is a middleware that implements authentication using HTTP Basic Authentication.
// The authenticate function argument must return nil to indicate that authentication succeeded.
// Any non-nil error value indicates that authentication failed.
// The WWW-Authenticate header will be set if the error value has status code 401 Unauthorized
// using the realm argument. If the realm argument is empty, the realm is set to the hostname.
//
// Note that basic authentication is only secure over HTTPS.
func BasicAuth(realm string, authenticate func(username, password string) error) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, _ := r.BasicAuth()
			if err := authenticate(username, password); err != nil {
				if httpsyproblem.StatusCode(err) == http.StatusUnauthorized {
					if w.Header().Get("WWW-Authenticate") == "" {
						realm := realm
						if realm == "" {
							realm = r.Host
						}
						w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s", charset="utf-8"`, realm))
					}
				}
				Error(w, r, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RouteParam is a middleware that extracts the head URL parameter
// from the URL path and stores it as a form value.
//
//  RouteParam("orderID") // create the middleware
//  RouteParamValue(r, "orderID") // get the parameter in the handler
//
// The URL parameter may optionally be given a pattern constraint
// that is matched using path.Match by adding a colon followed by the pattern:
//  RouteParam("myparam:?*") // matches any sequence of one or more characters
//
// It is also possible to use an empty name. In this case the pattern
// constraint is applied but the value is not stored in the form values:
//  RouteParam(":v[12]") // routes /v1 and /v2 to the same handler
func RouteParam(param string) func(http.Handler) http.Handler {
	name, pattern := param, "?*"

	if i := strings.Index(param, ":"); i >= 0 {
		name, pattern = param[:i], param[i+1:]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var head string
			r = cloneRequestURL(r)
			if head, r.URL.Path = ShiftPath(r.URL.Path); head == "" {
				Error(w, r, httpsyproblem.StatusNotFound)
				return
			} else if ok, _ := path.Match(pattern, head); !ok {
				Error(w, r, httpsyproblem.StatusNotFound)
				return
			} else if name != "" {
				r = setParamValue(r, name, head)
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

// SetErrorHandler is a middleware that sets the error handler used by Error.
func SetErrorHandler(h ErrorHandlerFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, WithContextValue(r, keyErrorHandlerCtxKey, h))
		})
	}
}

// Recoverer recovers from panics by responding with an HTTP 500 internal server error.
// The middleware does not recover from http.ErrAbortHandler.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil && v != http.ErrAbortHandler {
				switch err := v.(type) {
				case error:
					Error(w, r, err)
				case string:
					Error(w, r, fmt.Errorf(err))
				default:
					Error(w, r, fmt.Errorf("%v", err))
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// If applies the middlewares only if the condition is true.
func If(cond func(*http.Request) bool, then http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cond(r) {
				then.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
