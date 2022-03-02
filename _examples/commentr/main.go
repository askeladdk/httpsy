package main

import (
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/askeladdk/httpsy"
)

var funcMap = template.FuncMap{
	"timeFormat": func(t time.Time) string {
		return t.Format("Mon, 02 Jan 2006 15:04:05 MST")
	},
}

var indexTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`
<html>
	<head>
		<title>Commentr</title>
	</head>
	<body>
		<h1>Commentr</h1>
			<div>
				<form action="/" method="POST">
					Leave a message: <input type="text" name="message">
					<input type="submit" value="Submit">
					<input type="hidden" value="{{.CSRFToken}}" name="__csrf">
				</form>
			</div>

			{{ range .Posts }}
				<div>{{ .Time | timeFormat }}: {{ .Message }}</div>
			{{ end }}
	</body>
</html>
`))

type post struct {
	Message string
	Time    time.Time
}

type commentr struct {
	sync.RWMutex
	posts []post
}

func (s *commentr) renderPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Posts     []post
		CSRFToken string
	}{
		Posts:     s.posts,
		CSRFToken: w.Header().Get("x-csrf-token"),
	}
	renderer := httpsy.TemplateRenderer{Template: indexTemplate}
	httpsy.Render(renderer, w, r, http.StatusOK, data)
}

func (s *commentr) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.RLock()
		defer s.RUnlock()
		s.renderPage(w, r)
	case http.MethodPost:
		s.Lock()
		defer s.Unlock()
		message := r.FormValue("message")
		if message != "" {
			s.posts = append(s.posts, post{message, time.Now()})
		}
		s.renderPage(w, r)
	}
}

func main() {
	s := &commentr{}
	mux := http.NewServeMux()
	csrf := httpsy.CSRF{
		Secret:      "the eagle lands at midnight",
		FormKey:     "__csrf",
		SessionFunc: func(*http.Request) (string, bool) { return "", true },
		Expires:     24 * time.Hour,
	}
	mux.Handle("/", csrf.Handle(s))
	_ = http.ListenAndServe(":8080", mux)
}
