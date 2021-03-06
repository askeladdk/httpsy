package httpsy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/askeladdk/httpsy/httpsyproblem"
)

func TestProblemContentType(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Error(w, r, httpsyproblem.Wrap(nil, http.StatusBadRequest))
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	SetErrorHandler(ErrorHandlerFunc(JSONError))(endpoint).ServeHTTP(w, r)
	if w.Header().Get("Content-Type") != "application/problem+json; charset=utf-8" {
		t.Fatal()
	}
}

func TestContextKeyTypeOf(t *testing.T) {
	var k1 contextKey = "test"
	var k2 string = "test"
	r := httptest.NewRequest("GET", "/", nil)
	r = SetContextValue(r, k1, "world")
	r = SetContextValue(r, k2, "WORLD")
	v1 := GetContextValue(r, k1)
	v2 := GetContextValue(r, k2)
	if v1 == v2 {
		t.Fatal()
	}
}
