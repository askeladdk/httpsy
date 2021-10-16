package httpsy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

func TestParam(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s", ParamValue(r, "a"))
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

	unauthorized := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Error(w, r, httpsyproblem.StatusUnauthorized)
		})
	}

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

func TestWithHeader(t *testing.T) {
	endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	WithHeader("Server", "Gopher")(endpoint).ServeHTTP(w, r)
	if w.Header().Get("Server") != "Gopher" {
		t.Fatal()
	}
}

func TestTimeout(t *testing.T) {
	t.Run("200", func(t *testing.T) {
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			_, _ = io.WriteString(w, "hello")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(3*time.Second, nil)(endpoint).ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatal()
		}

		if !bytes.HasPrefix(w.Body.Bytes(), []byte("hello")) {
			t.Fatal()
		}
	})

	t.Run("503", func(t *testing.T) {
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			_, _ = io.WriteString(w, "hello")
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(1*time.Second, nil)(endpoint).ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusServiceUnavailable {
			t.Fatal()
		}

		if bytes.HasPrefix(w.Body.Bytes(), []byte("hello")) {
			t.Fatal()
		}
	})

	t.Run("panic", func(t *testing.T) {
		defer func() {
			if v := recover(); v != io.EOF {
				t.Fatal()
			}
		}()

		endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(io.EOF)
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(1*time.Second, nil)(endpoint).ServeHTTP(w, r)
	})

	t.Run("cancel", func(t *testing.T) {
		endpoint := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for {
				select {
				case <-r.Context().Done():
					return
				default:
					_, _ = io.WriteString(w, "hello")
				}
			}
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		Timeout(1*time.Second, nil)(endpoint).ServeHTTP(w, r)
		if w.Result().StatusCode != http.StatusServiceUnavailable {
			t.Fatal()
		}

		if bytes.HasPrefix(w.Body.Bytes(), []byte("hello")) {
			t.Fatal()
		}
	})
}
