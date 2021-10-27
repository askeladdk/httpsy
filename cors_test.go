package httpsy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func assertHeaders(t *testing.T, h http.Header, expected map[string]string) {
	for k, v := range expected {
		if h.Get(k) != v {
			t.Fatal(k, h.Get(k), v)
		}
	}
}

func TestCORS(t *testing.T) {
	endpoint := GetHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	cors := CORS{
		AllowHeaders: []string{"X-Requested-With"},
		AllowMethods: []string{"GET", "OPTIONS"},
		AllowOrigins: []string{"https://example.com"},
		MaxAge:       24 * 60 * 60,
	}

	x := cors.Handle(Methods(endpoint))

	t.Run("no-origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)

		x.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Fatal()
		}

		assertHeaders(t, w.Header(), map[string]string{
			"Access-Control-Allow-Origin": "",
			"Vary":                        "",
		})
	})

	t.Run("request-success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Origin", "https://example.com")

		x.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Fatal()
		}

		assertHeaders(t, w.Header(), map[string]string{
			"Access-Control-Allow-Origin": "https://example.com",
			"Vary":                        "Origin",
		})
	})

	t.Run("request-fail", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Origin", "https://evil.com")

		x.ServeHTTP(w, r)

		if w.Code != http.StatusNoContent {
			t.Fatal()
		}

		assertHeaders(t, w.Header(), map[string]string{
			"Access-Control-Allow-Origin": "",
		})
	})

	t.Run("preflight-success", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("OPTIONS", "/", nil)
		r.Header.Set("Origin", "https://example.com")
		r.Header.Set("X-Requested-With", "xhr")
		r.Header.Set("Access-Control-Request-Headers", "X-Requested-With")
		r.Header.Set("Access-Control-Request-Method", "GET")

		x.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatal()
		}

		assertHeaders(t, w.Header(), map[string]string{
			"Access-Control-Allow-Origin":  "https://example.com",
			"Access-Control-Allow-Headers": "X-Requested-With",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
			"Access-Control-Max-Age":       "86400",
			"Content-Length":               "0",
		})
	})

	t.Run("preflight-fail", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("OPTIONS", "/", nil)
		r.Header.Set("Origin", "https://evil.com")
		r.Header.Set("X-Requested-With", "XMLHttpRequest")
		r.Header.Set("Access-Control-Request-Headers", "X-Requested-With")
		r.Header.Set("Access-Control-Request-Method", "DELETE")

		x.ServeHTTP(w, r)

		if w.Code != http.StatusMethodNotAllowed {
			t.Fatal()
		}

		assertHeaders(t, w.Header(), map[string]string{
			"Access-Control-Allow-Origin": "",
		})
	})
}

func TestCORSDefaults(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	x := new(CORS).Handle(endpoint)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/", nil)
	r.Header.Set("Origin", "https://evil.com")
	r.Header.Set("X-Requested-With", "XMLHttpRequest")
	r.Header.Set("Access-Control-Request-Headers", "X-Requested-With, Content-Type")
	r.Header.Set("Access-Control-Request-Method", "POST")

	x.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatal()
	}

	assertHeaders(t, w.Header(), map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Headers": "X-Requested-With, Content-Type",
		"Access-Control-Allow-Methods": "POST",
		"Access-Control-Max-Age":       "-1",
		"Content-Length":               "0",
	})
}
