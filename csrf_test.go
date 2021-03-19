package httpsy

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCSRFMaskUnmask(t *testing.T) {
	buf := make([]byte, 2*csrfTokenLength)
	token := make([]byte, csrfTokenLength)
	secret := make([]byte, csrfTokenLength)
	randomNoise(token)
	randomNoise(secret)
	copy(buf[csrfTokenLength:], token)
	csrfMask(secret, buf)
	csrfUnmask(secret, buf)
	if !bytes.Equal(token, buf[csrfTokenLength:]) {
		t.Fatal()
	}
}

func TestCSRFVerify(t *testing.T) {
	b64 := base64.URLEncoding
	sbuf := make([]byte, 2*csrfTokenLength)
	token := make([]byte, csrfTokenLength)
	secret := make([]byte, csrfTokenLength)
	randomNoise(token)
	randomNoise(secret)
	copy(sbuf[csrfTokenLength:], token)
	csrfMask(secret, sbuf)
	bt := b64.EncodeToString(token)
	bs := b64.EncodeToString(sbuf)
	if !csrfVerify(secret, bt, bs) {
		t.Fatal()
	}
}

func TestCSRFRequests(t *testing.T) {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		token, _ := GetCSRFToken(r)
		fmt.Fprintf(w, "%s", token)
	}

	csrf := CSRF{
		FieldName:  "csrf",
		CookieName: "csrf",
	}
	randomNoise(csrf.Secret[:])

	x := csrf.Handler(http.HandlerFunc(endpoint))

	var cookie *http.Cookie
	var token string

	// get the cookie and token
	t.Run("GET", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		x.ServeHTTP(w, r)
		cookie = w.Result().Cookies()[0]
		token = w.Body.String()
		if w.Code != 200 || cookie.Name == "" || cookie.Value == "" {
			t.Fatal()
		}
	})

	// post without token must fail
	t.Run("POST-forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		r.AddCookie(cookie)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatal()
		}
	})

	// post with token in form must succeed
	t.Run("POST-form", func(t *testing.T) {
		w := httptest.NewRecorder()
		form := url.Values{}
		form.Add("csrf", token)
		r := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		r.Header.Set("content-type", "application/x-www-form-urlencoded")
		r.AddCookie(cookie)
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
		r.AddCookie(cookie)
		x.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal()
		}
	})
}
