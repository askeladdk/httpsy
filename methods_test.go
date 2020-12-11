package httpsy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMethods(t *testing.T) {
	methods := []struct {
		Method  string
		Handler http.Handler
	}{
		{http.MethodConnect, Methods(ConnectHandlerFunc(Pong))},
		{http.MethodDelete, Methods(DeleteHandlerFunc(Pong))},
		{http.MethodGet, Methods(GetHandlerFunc(Pong))},
		{http.MethodHead, Methods(HeadHandlerFunc(Pong))},
		{http.MethodOptions, Methods(OptionsHandlerFunc(Pong))},
		{http.MethodPatch, Methods(PatchHandlerFunc(Pong))},
		{http.MethodPost, Methods(PostHandlerFunc(Pong))},
		{http.MethodPut, Methods(PutHandlerFunc(Pong))},
		{http.MethodTrace, Methods(TraceHandlerFunc(Pong))},
	}

	for i, m := range methods {
		t.Run(m.Method+"_204", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(m.Method, "/", nil)
			m.Handler.ServeHTTP(w, r)
			if w.Code != http.StatusNoContent {
				t.Fatal()
			}
		})

		t.Run(m.Method+"_405", func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(methods[(i+1)%len(methods)].Method, "/", nil)
			m.Handler.ServeHTTP(w, r)
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatal()
			}
		})
	}

	t.Run("TEAPOT", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("TEAPOT", "/", nil)
		Methods(http.HandlerFunc(Pong)).ServeHTTP(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatal()
		}
	})
}

type testMethodHandler struct{ MethodHandler }

func (m testMethodHandler) ServeGet(w http.ResponseWriter, r *http.Request) {}

func TestMethodHandler(t *testing.T) {
	x := Methods(testMethodHandler{})

	t.Run("GET", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatal()
		}
	})

	t.Run("POST", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != http.StatusMethodNotAllowed {
			t.Fatal()
		}
	})
}

func TestGetHead(t *testing.T) {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Gopher")
	}

	x := Methods(GetHeadHandlerFunc(endpoint))

	t.Run("GET", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.Len() != 6 {
			t.Fatal()
		}
	})

	t.Run("HEAD", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("HEAD", "/", nil)
		x.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.Len() != 0 {
			t.Fatal()
		}
	})
}
