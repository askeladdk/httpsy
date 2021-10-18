package httpsy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCSRFRequests(t *testing.T) {
	endpoint := func(w http.ResponseWriter, r *http.Request) {}

	csrf := CSRF{
		Secret:      "my secret key",
		FormKey:     "csrf-form-key",
		Expires:     10 * time.Minute,
		SessionFunc: func(_ *http.Request) (string, bool) { return "a", true },
	}

	x := csrf.Handle(http.HandlerFunc(endpoint))

	var token string

	// get the token
	t.Run("GET", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		x.ServeHTTP(w, r)
		token = w.Header().Get("x-csrf-token")
		if w.Code != 200 || token == "" {
			t.Fatal()
		}
	})

	// post without token must fail
	t.Run("POST-forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal()
		}
	})

	// post with token in form must succeed
	t.Run("POST-form", func(t *testing.T) {
		w := httptest.NewRecorder()
		form := url.Values{}
		form.Add("csrf-form-key", token)
		r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		r.Header.Set("content-type", "application/x-www-form-urlencoded")
		x.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal()
		}
	})

	// post with token in header must succeed
	t.Run("POST-header", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("x-csrf-token", token)
		x.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal()
		}
	})

	// http->https mitm must fail
	t.Run("POST-mitm", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("x-csrf-token", token)
		r.Header.Set("referer", "http://example.com")
		r.URL.Scheme, r.URL.Host = "https", "example.com"
		x.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal()
		}
	})
}
