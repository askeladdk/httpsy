package httpsytrace_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/askeladdk/httpsy/httpsytrace"
)

type metrics struct {
	httpsytrace.DefaultTrace
	BytesWritten int64
	StatusCode   int
}

func (m *metrics) WriteHeader(w http.ResponseWriter, statusCode int) {
	m.StatusCode = statusCode
	m.DefaultTrace.WriteHeader(w, statusCode)
}

func (m *metrics) Write(w io.Writer, p []byte) (int, error) {
	m.BytesWritten += int64(len(p))
	return m.DefaultTrace.Write(w, p)
}

// Create a logger middleware to log all requests and their metrics.
func Example_logging() {
	logger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := metrics{}
			next.ServeHTTP(httpsytrace.Wrap(w, &m), r)
			fmt.Printf("%s %s %d %d\n", r.Method, r.URL, m.StatusCode, m.BytesWritten)
		})
	}

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world!")
	})

	mux := http.NewServeMux()
	mux.Handle("/", logger(endpoint))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, r)
	// Output: GET / 200 13
}
