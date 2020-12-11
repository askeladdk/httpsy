package httpsytrace_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/askeladdk/httpsy"
	"github.com/askeladdk/httpsy/httpsytrace"
)

// Create a logger middleware to log all requests and their metrics.
func Example_logging() {
	logger := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := httpsytrace.Metrics{}
			next.ServeHTTP(httpsytrace.Hook(w, &m), r)
			fmt.Printf("%s %s %d %d\n", r.Method, r.URL, m.StatusCode, m.BytesWritten)
		})
	}

	mux := httpsy.NewServeMux()
	mux.Use(logger)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world!")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, r)
	// Output: GET / 200 13
}
