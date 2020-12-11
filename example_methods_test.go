package httpsy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/askeladdk/httpsy"
)

type Resource struct{ httpsy.MethodHandler }

func (res *Resource) ServeGet(w http.ResponseWriter, r *http.Request)  {}
func (res *Resource) ServePost(w http.ResponseWriter, r *http.Request) {}

// GET and POST requests will return 200 OK but all other
// request methods result in 405 method not allowed.
func ExampleMethodHandler() {
	endpoint := &Resource{}
	mux := httpsy.NewServeMux()
	mux.Handle("/", endpoint)

	var w *httptest.ResponseRecorder
	var r *http.Request

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)

	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)

	w = httptest.NewRecorder()
	r = httptest.NewRequest("PUT", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)

	// Output:
	// 200
	// 200
	// 405
}
