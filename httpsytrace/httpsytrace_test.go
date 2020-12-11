package httpsytrace

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testTracer(status, written *int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := Metrics{}
			m.Start = time.Now()
			next.ServeHTTP(Hook(w, &m), r)
			fmt.Printf("%s %s %d %d %v\n", r.Method, r.URL, m.StatusCode, m.BytesWritten, m.Duration())
			*status = m.StatusCode
			*written = int(m.BytesWritten)
		})
	}
}

func TestMetricsTrace(t *testing.T) {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		b := bytes.NewBuffer(make([]byte, 4096))
		io.Copy(w, b)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	var status, written int
	x := testTracer(&status, &written)(http.HandlerFunc(endpoint))
	x.ServeHTTP(w, r)
	if status != 201 || status != w.Code || written != w.Body.Len() {
		t.Fatal()
	}
}
