package main

import (
	"fmt"
	"net/http"

	"github.com/askeladdk/httpsy"
)

func main() {
	mux := httpsy.NewServeMux()
	mux.Handle("/", httpsy.GetHeadHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, world!")
	}))
	http.ListenAndServe(":8080", mux)
}
