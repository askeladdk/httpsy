package httpsy_test

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
	httpsy.MethodHandler
	sync.RWMutex
	posts []post
}

func (s *commentr) ServeGet(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	defer s.RUnlock()
	data := struct{ Posts []post }{s.posts}
	renderer := httpsy.TemplateRenderer{indexTemplate, ""}
	httpsy.Render(w, r, http.StatusOK, data, renderer)
}

func (s *commentr) ServePost(w http.ResponseWriter, r *http.Request) {
	s.Lock()
	defer s.Unlock()

	message := r.FormValue("message")

	if message != "" {
		s.posts = append(s.posts, post{message, time.Now()})
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// This example demonstrates a tiny but functional page where users can leave comments.
// Navigate to http://localhost:8080 to leave a comment.
func Example_restful() {
	s := &commentr{}
	mux := httpsy.NewServeMux()
	mux.Handle("/", s)
	http.ListenAndServe(":8080", mux)
}
