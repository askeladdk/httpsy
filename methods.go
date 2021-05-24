package httpsy

import (
	"net/http"
	"strings"
)

// method handler interfaces idea taken from
// https://themue.dev/blog/2019/03/23/dont-be-afraid-of-multiplexing/

// MethodHandler is a helper intended to be used by embedding it in another struct
// that implements one or more method handler interfaces (GetHandler, PostHandler, etc).
// MethodHandler itself implements http.Handler and replies to all request methods
// that have not been implemented with HTTP 405 method not allowed.
type MethodHandler struct{}

// ServeHTTP implements http.Handler.
func (m MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Error(w, r, StatusMethodNotAllowed)
}

// ConnectHandler replies to CONNECT requests.
type ConnectHandler interface {
	ServeConnect(w http.ResponseWriter, r *http.Request)
}

// DeleteHandler replies to DELETE requests.
type DeleteHandler interface {
	ServeDelete(w http.ResponseWriter, r *http.Request)
}

// GetHandler replies to GET requests.
type GetHandler interface {
	ServeGet(w http.ResponseWriter, r *http.Request)
}

// HeadHandler replies to HEAD requests.
type HeadHandler interface {
	ServeHead(w http.ResponseWriter, r *http.Request)
}

// OptionsHandler replies to OPTIONS requests.
type OptionsHandler interface {
	ServeOptions(w http.ResponseWriter, r *http.Request)
}

// PatchHandler replies to PATCH requests.
type PatchHandler interface {
	ServePatch(w http.ResponseWriter, r *http.Request)
}

// PostHandler replies to POST requests.
type PostHandler interface {
	ServePost(w http.ResponseWriter, r *http.Request)
}

// PutHandler replies to PUT requests.
type PutHandler interface {
	ServePut(w http.ResponseWriter, r *http.Request)
}

// TraceHandler replies to TRACE requests.
type TraceHandler interface {
	ServeTrace(w http.ResponseWriter, r *http.Request)
}

// ConnectHandlerFunc adapts an ordinary HTTP handler to a ConnectHandler.
type ConnectHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeConnect implements ConnectHandler.
func (f ConnectHandlerFunc) ServeConnect(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f ConnectHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodConnect)
	Error(w, r, StatusMethodNotAllowed)
}

// DeleteHandlerFunc adapts an ordinary HTTP handler to a DeleteHandler.
type DeleteHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeDelete implements DeleteHandler.
func (f DeleteHandlerFunc) ServeDelete(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f DeleteHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodDelete)
	Error(w, r, StatusMethodNotAllowed)
}

// HeadHandlerFunc adapts an ordinary HTTP handler to a HeadHandler.
type HeadHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeHead implements HeadHandler.
func (f HeadHandlerFunc) ServeHead(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f HeadHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodHead)
	Error(w, r, StatusMethodNotAllowed)
}

// GetHandlerFunc adapts an ordinary HTTP handler to a GetHandler.
type GetHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeGet implements GetHandler.
func (f GetHandlerFunc) ServeGet(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f GetHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodGet)
	Error(w, r, StatusMethodNotAllowed)
}

// OptionsHandlerFunc adapts an ordinary HTTP handler to a OptionsHandler.
type OptionsHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeOptions implements OptionsHandler.
func (f OptionsHandlerFunc) ServeOptions(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f OptionsHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodOptions)
	Error(w, r, StatusMethodNotAllowed)
}

// PatchHandlerFunc adapts an ordinary HTTP handler to a PatchHandler.
type PatchHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServePatch implements OptionsHandler.
func (f PatchHandlerFunc) ServePatch(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f PatchHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodPatch)
	Error(w, r, StatusMethodNotAllowed)
}

// PostHandlerFunc adapts an ordinary HTTP handler to a PostHandler.
type PostHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServePost implements PostHandler.
func (f PostHandlerFunc) ServePost(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f PostHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodPost)
	Error(w, r, StatusMethodNotAllowed)
}

// PutHandlerFunc adapts an ordinary HTTP handler to a PutHandler.
type PutHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServePut implements PutHandler.
func (f PutHandlerFunc) ServePut(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f PutHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodPut)
	Error(w, r, StatusMethodNotAllowed)
}

// TraceHandlerFunc adapts an ordinary HTTP handler to a TraceHandler.
type TraceHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeTrace implements TraceHandler.
func (f TraceHandlerFunc) ServeTrace(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHTTP implements http.Handler.
func (f TraceHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", http.MethodTrace)
	Error(w, r, StatusMethodNotAllowed)
}

type discardResponseWriter struct{ http.ResponseWriter }

func (w discardResponseWriter) Write(p []byte) (int, error) { return len(p), nil }

// GetHeadHandlerFunc adapts an ordinary HTTP handler to a GetHandler and HeadHandler.
// GET and HEAD requests are both directed to the handler but HEAD requests will
// not send the response body even if it is written to.
// The handler is responsible for skipping expensive actions when it receives a HEAD request.
type GetHeadHandlerFunc func(w http.ResponseWriter, r *http.Request)

// ServeGet implements GetHandler.
func (f GetHeadHandlerFunc) ServeGet(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// ServeHead implements HeadHandler.
func (f GetHeadHandlerFunc) ServeHead(w http.ResponseWriter, r *http.Request) {
	f(discardResponseWriter{w}, r)
}

// ServeHTTP implements http.Handler.
func (f GetHeadHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD")
	Error(w, r, StatusMethodNotAllowed)
}

func isMethodHandler(handler http.Handler) bool {
	switch handler.(type) {
	case ConnectHandler:
		return true
	case DeleteHandler:
		return true
	case GetHandler:
		return true
	case HeadHandler:
		return true
	case OptionsHandler:
		return true
	case PatchHandler:
		return true
	case PostHandler:
		return true
	case PutHandler:
		return true
	case TraceHandler:
		return true
	default:
		return false
	}
}

func listAllowedMethods(handler http.Handler) (allowed []string) {
	if _, ok := handler.(ConnectHandler); ok {
		allowed = append(allowed, http.MethodConnect)
	}
	if _, ok := handler.(DeleteHandler); ok {
		allowed = append(allowed, http.MethodDelete)
	}
	if _, ok := handler.(GetHandler); ok {
		allowed = append(allowed, http.MethodGet)
	}
	if _, ok := handler.(HeadHandler); ok {
		allowed = append(allowed, http.MethodHead)
	}
	if _, ok := handler.(OptionsHandler); ok {
		allowed = append(allowed, http.MethodOptions)
	}
	if _, ok := handler.(PatchHandler); ok {
		allowed = append(allowed, http.MethodPatch)
	}
	if _, ok := handler.(PostHandler); ok {
		allowed = append(allowed, http.MethodPost)
	}
	if _, ok := handler.(PutHandler); ok {
		allowed = append(allowed, http.MethodPut)
	}
	if _, ok := handler.(TraceHandler); ok {
		allowed = append(allowed, http.MethodTrace)
	}
	return
}

// Methods is a middleware that adapts the handler to dispatch to any implemented (Method)Handlers.
// Due to the nature of middlewares, it must be the first middleware to be applied to the endpoint.
// It is automatically applied by ServeMux and usually does not need to be called manually.
func Methods(next http.Handler) http.Handler {
	if !isMethodHandler(next) {
		return next
	}

	allowedMethods := listAllowedMethods(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodConnect:
			if h, ok := next.(ConnectHandler); ok {
				h.ServeConnect(w, r)
				return
			}
		case http.MethodDelete:
			if h, ok := next.(DeleteHandler); ok {
				h.ServeDelete(w, r)
				return
			}
		case http.MethodHead:
			if h, ok := next.(HeadHandler); ok {
				h.ServeHead(w, r)
				return
			}
		case http.MethodGet:
			if h, ok := next.(GetHandler); ok {
				h.ServeGet(w, r)
				return
			}
		case http.MethodOptions:
			if h, ok := next.(OptionsHandler); ok {
				h.ServeOptions(w, r)
				return
			}
		case http.MethodPatch:
			if h, ok := next.(PatchHandler); ok {
				h.ServePatch(w, r)
				return
			}
		case http.MethodPost:
			if h, ok := next.(PostHandler); ok {
				h.ServePost(w, r)
				return
			}
		case http.MethodPut:
			if h, ok := next.(PutHandler); ok {
				h.ServePut(w, r)
				return
			}
		case http.MethodTrace:
			if h, ok := next.(TraceHandler); ok {
				h.ServeTrace(w, r)
				return
			}
		}

		w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
		next.ServeHTTP(w, r)
	})
}
