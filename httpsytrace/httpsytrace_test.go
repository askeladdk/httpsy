package httpsytrace

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
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

type writeTracerFunc func([]byte) (int, error)

func (f writeTracerFunc) Write(w io.Writer, p []byte) (int, error) {
	if n, err := f(p); err != nil {
		return n, err
	}
	return w.Write(p)
}

func (f writeTracerFunc) WriteHeader(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
}

func (f writeTracerFunc) Flush(flusher http.Flusher) {
	flusher.Flush()
}

func (f writeTracerFunc) Hijack(hijacker http.Hijacker) (net.Conn, *bufio.ReadWriter, error) {
	return hijacker.Hijack()
}

func (f writeTracerFunc) Push(pusher http.Pusher, target string, opts *http.PushOptions) error {
	return pusher.Push(target, opts)
}

type mockReadFromRecorder struct {
	*httptest.ResponseRecorder
}

func (w *mockReadFromRecorder) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(w.ResponseRecorder, r)
}

func TestReadFrom(t *testing.T) {
	var body []byte
	for i := 0; i < 10000; i++ {
		body = append(body, []byte("gopher")...)
	}

	sum := md5.Sum(body)
	expected := hex.EncodeToString(sum[:])
	var got string

	tracer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := md5.New()
			next.ServeHTTP(Hook(w, writeTracerFunc(h.Write)), r)
			got = hex.EncodeToString(h.Sum(nil))
		})
	}

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(io.ReaderFrom).ReadFrom(bytes.NewReader(body))
	})

	w := mockReadFromRecorder{httptest.NewRecorder()}
	r := httptest.NewRequest("GET", "/", nil)
	tracer(endpoint).ServeHTTP(&w, r)

	bodysum := md5.Sum(w.Body.Bytes())
	bodygot := hex.EncodeToString(bodysum[:])

	if expected != got || expected != bodygot || len(w.Body.Bytes()) != len(body) {
		t.Fatal()
	}
}
