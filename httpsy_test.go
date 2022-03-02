package httpsy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/askeladdk/httpsyproblem"
)

func TestProblemContentType(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, httpsyproblem.StatusBadRequest)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Add("Accept", "application/json")
	endpoint.ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "application/problem+json; charset=utf-8" {
		t.Fatal()
	}
}

func TestContextKeyTypeOf(t *testing.T) {
	var k1 = keyErrorHandlerCtxKey
	var k2 = paramMapCtxKey
	if k1 == k2 {
		t.Fatal("k1 == k2")
	}
	r := httptest.NewRequest("GET", "/", nil)
	r = WithContextValue(r, k1, "world")
	r = WithContextValue(r, k2, "WORLD")
	v1 := r.Context().Value(k1)
	v2 := r.Context().Value(k2)
	if v1 == v2 {
		t.Fatal()
	}
}
