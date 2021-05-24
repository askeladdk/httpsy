package httpsy_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/askeladdk/httpsy"
)

// This example demonstrates routing based on an URL parameter and GET request.
func Example_hello() {
	hello := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %s!", r.FormValue("you"))
	}

	mux := httpsy.NewServeMux()
	mux.HandleParam("/", "you", httpsy.GetHandlerFunc(hello))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Gopher", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output: Hello, Gopher!
}

// This example demonstrates how to route multiple URL parameters using a nested ServeMux.
// ServeMux can be nested multiple times using Route or RouteParam.
func Example_routeParam() {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s orbits %s", r.FormValue("planet"), r.FormValue("sun"))
	}

	mux := httpsy.NewServeMux()
	mux.RouteParam("/sun/", "sun", func(mux *httpsy.ServeMux) {
		mux.HandleParamFunc("/planet/", "planet", endpoint)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/sun/sol/planet/mars", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output: mars orbits sol
}

// Respond to all requests with HTTP 418 I'm a teapot.
func ExampleError() {
	endpoint := func(w http.ResponseWriter, r *http.Request) {
		httpsy.Error(w, r, httpsy.StatusTeapot)
	}
	mux := httpsy.NewServeMux()
	mux.HandleFunc("/", endpoint)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Body.String())
	// Output: I'm a teapot
}

// This example demonstrates how to use httpsy.If to
// selectively apply middleware based on a predicate.
func ExampleIf() {
	// Define a predicate function that only accepts POST requests.
	isPost := httpsy.PredicateFunc(func(r *http.Request) bool {
		return r.Method == http.MethodPost
	})

	// Define an authenticator that forbids all authentication attempts.
	authFunc := httpsy.AuthenticatorFunc(func(r *http.Request) (*http.Request, error) {
		return r, httpsy.StatusForbidden
	})

	// Define the ServeMux and apply the If middleware
	// to restrict authentication to POST requests.
	mux := httpsy.NewServeMux()
	mux.Use(httpsy.If(isPost, httpsy.Authenticate(authFunc)))
	mux.HandleFunc("/", httpsy.Pong)

	// GET request succeeds.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)

	// POST request is forbidden.
	w = httptest.NewRecorder()
	r = httptest.NewRequest("POST", "/", nil)
	mux.ServeHTTP(w, r)
	fmt.Println(w.Code)

	// Output:
	// 204
	// 403
}
