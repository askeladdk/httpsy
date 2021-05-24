package httpsy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServeMux(t *testing.T) {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "planet %s orbits %s", r.FormValue("planet"), r.FormValue("system"))
	}

	mux := NewServeMux()
	mux.Use(Recoverer)
	mux.Route("/", func(mux *ServeMux) {
		mux.RouteParam("/system/", "system", func(mux *ServeMux) {
			mux.HandleParamFunc("/", "planet", endpoint)
		})
		mux.HandleFunc("/other", func(w http.ResponseWriter, r *http.Request) {})
	})

	t.Run("planet", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/system/sol/mars", nil)
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK || w.Body.String() != "planet mars orbits sol" {
			t.Fatal()
		}
	})

	t.Run("other", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/other", nil)
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatal()
		}

	})
}

func TestServeMux404(t *testing.T) {
	mux := NewServeMux()
	mux.Use(SetErrorHandler(ErrorHandlerFunc(JSONError)))
	mux.Route("/test/", func(mux *ServeMux) {})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test/q", nil)
	mux.ServeHTTP(w, r)
	s := w.Body.String()
	if w.Code != http.StatusNotFound || s != "{\"status\":404,\"title\":\"Not Found\"}\n" {
		t.Fatal()
	}
}
