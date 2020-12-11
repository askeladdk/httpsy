package httpsy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	usernameKey := NewContextKey("username")

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := GetContextValue(r, usernameKey).(string)
		fmt.Fprintf(w, "%s", username)
	})

	basicAuth := AuthenticatorFunc(func(r *http.Request) (*http.Request, error) {
		if username, password, ok := r.BasicAuth(); !ok {
			return r, ErrAuthenticationFailed
		} else if username == "gopher" && password == "secret" {
			return SetContextValue(r, usernameKey, username), nil
		} else if username == "java" {
			return r, ErrAccessForbidden
		}
		return r, ErrAuthenticationFailed
	})

	x := Authenticate(basicAuth)(endpoint)

	t.Run("200", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.SetBasicAuth("gopher", "secret")
		x.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Body.String() != "gopher" {
			t.Fatal()
		}
	})

	t.Run("401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.SetBasicAuth("gopher", "helloworld")
		x.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatal()
		}
	})

	t.Run("403", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		r.SetBasicAuth("java", "secret")
		x.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatal()
		}
	})
}

func TestParam(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", r.FormValue("a"))
	})

	x := Param("a:[a-z]+$")(endpoint)

	t.Run("200", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/hello", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Body.String() != "hello" {
			t.Fatal()
		}
	})

	t.Run("404", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/hell0", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusNotFound {
			t.Fatal()
		}
	})
}

func TestChainNoCacheRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	x := Chain(NoCache, RequestID).Handler(http.HandlerFunc(Pong))
	x.ServeHTTP(w, r)
	if w.Header().Get("Expires") == "" || r.Header.Get("X-Request-ID") == "" {
		t.Fatal()
	}
}

func TestIfEndPoint(t *testing.T) {
	isPost := PredicateFunc(func(r *http.Request) bool { return r.Method == "POST" })

	x := IfChain(isPost, EndpointFunc(Unauthorized))(http.HandlerFunc(Pong))

	t.Run("204", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatal()
		}
	})

	t.Run("401", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatal()
		}
	})
}

func TestSetErrorHandlerPanic(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("gopher!")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	x := Chain(SetErrorHandler(ErrorHandlerFunc(JSONError)), Recoverer).Handler(endpoint)
	x.ServeHTTP(w, r)

	s := w.Body.String()
	if w.Code != http.StatusInternalServerError || s != "{\"message\":\"gopher!\",\"status\":500}\n" {
		t.Fatal()
	}
}
