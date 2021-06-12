package httpsy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate(t *testing.T) {
	var usernameKey contextKey = "username"

	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username := GetContextValue(r, usernameKey).(string)
		fmt.Fprintf(w, "%s", username)
	})

	basicAuth := func(r *http.Request) (*http.Request, error) {
		if username, password, ok := r.BasicAuth(); !ok {
			return r, StatusUnauthorized
		} else if username == "gopher" && password == "secret" {
			return SetContextValue(r, usernameKey, username), nil
		} else if username == "java" {
			return r, StatusForbidden
		}
		return r, StatusUnauthorized
	}

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
		fmt.Fprintf(w, "%s", GetParamValue(r, "a"))
	})

	x := Param("a:*[a-z]")(endpoint)

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
	x := Chain(NoCache, RequestID).Handler(http.HandlerFunc(NoContent))
	x.ServeHTTP(w, r)
	if w.Header().Get("Expires") == "" || r.Header.Get("X-Request-ID") == "" {
		t.Fatal()
	}
}

func TestIfEndPoint(t *testing.T) {
	isPost := func(r *http.Request) bool { return r.Method == "POST" }

	unauthorized := EndpointFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, StatusUnauthorized)
	})

	x := IfChain(isPost, unauthorized)(http.HandlerFunc(NoContent))

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
	if w.Code != http.StatusInternalServerError || s != "{\"detail\":\"gopher!\",\"status\":500,\"title\":\"Internal Server Error\"}\n" {
		t.Fatal()
	}
}

func TestRecovererErrAbortHandler(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	t.Run("run", func(t *testing.T) {
		defer func() {
			if v := recover(); v != http.ErrAbortHandler {
				t.Fatal()
			}
		}()
		Recoverer(endpoint).ServeHTTP(w, r)
	})
}
