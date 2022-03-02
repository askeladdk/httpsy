package httpsy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/askeladdk/httpsyproblem"
)

func TestAuthenticate(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := r.BasicAuth()
		fmt.Fprintf(w, "%s", username)
	})

	authenticate := func(username, password string) error {
		if username == "gopher" && password == "secret" {
			return nil
		} else if username == "java" {
			return httpsyproblem.StatusForbidden
		}
		return httpsyproblem.StatusUnauthorized
	}

	x := BasicAuth("", authenticate)(endpoint)

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

func TestRouteParam(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", RouteParamValue(r, "a"))
	})

	x := RouteParam("a:*[a-z]")(endpoint)

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

func TestIfEndPoint(t *testing.T) {
	isPost := func(r *http.Request) bool { return r.Method == "POST" }

	unauthorized := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, httpsyproblem.StatusUnauthorized)
	})

	x := If(isPost, unauthorized)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

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

func TestRecoverer(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("gopher!")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	x := Recoverer(endpoint)
	x.ServeHTTP(w, r)

	s := w.Body.String()
	if w.Code != http.StatusInternalServerError || s != "Internal Server Error\n" {
		t.Fatal()
	}
}

func TestRecovererErrAbortHandler(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(fmt.Errorf("panic!"))
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	t.Run("run", func(t *testing.T) {
		defer func() {
			if v := recover(); v != nil {
				t.Fatal()
			}
		}()
		Recoverer(endpoint).ServeHTTP(w, r)
	})
}
