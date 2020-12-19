package httpsytrace

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkHandlerBaseline(b *testing.B) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(w, r)
	}
	b.StopTimer()
}

func BenchmarkHandlerServerTrace(b *testing.B) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(Wrap(w, ServerTrace{}), r)
		})
	}

	h := middleware(endpoint)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(w, r)
	}
	b.StopTimer()
}

func BenchmarkReaderFromBaseline(b *testing.B) {
	bs := make([]byte, 32*1024)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buffer := bytes.NewBuffer(bs)
		w.(io.ReaderFrom).ReadFrom(buffer)
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := mockReadFromRecorder{httptest.NewRecorder()}
		r := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(&w, r)
	}
	b.StopTimer()
}

func BenchmarkReaderFromServerTrace(b *testing.B) {
	bs := make([]byte, 32*1024)

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buffer := bytes.NewBuffer(bs)
		w.(io.ReaderFrom).ReadFrom(buffer)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(Wrap(w, ServerTrace{}), r)
		})
	}

	h := middleware(endpoint)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := mockReadFromRecorder{httptest.NewRecorder()}
		r := httptest.NewRequest("GET", "/", nil)
		h.ServeHTTP(&w, r)
	}
	b.StopTimer()
}
