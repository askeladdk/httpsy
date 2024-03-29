package httpsy

import (
	"net/http"
	"net/url"
	"path"
	"strings"
)

var (
	keyErrorHandlerCtxKey = &struct{ byte }{}
	paramMapCtxKey        = &struct{ byte }{}
)

func cloneRequestURL(r *http.Request) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	return r2
}

func stringsMatch(patterns []string, v string) bool {
	for _, pattern := range patterns {
		if ok, _ := path.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

func stringsJoinMap(elems []string, sep string, fn func(string) string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(fn(s))
	}
	return b.String()
}

func sameOrigin(url1, url2 *url.URL) bool {
	return url1 != nil && url2 != nil && url1.Scheme == url2.Scheme && url1.Host == url2.Host
}

func sourceOrigin(r *http.Request, fallback *url.URL) *url.URL {
	if origin := r.Header.Get("Origin"); origin != "" {
		u, _ := url.Parse(origin)
		return u
	} else if referer := r.Header.Get("Referer"); referer != "" {
		u, _ := url.Parse(referer)
		return u
	}
	return fallback
}

func targetOrigin(r *http.Request, fallback *url.URL) *url.URL {
	if xfh := r.Header.Get("X-Forwarded-Host"); xfh != "" {
		u, _ := url.Parse(xfh)
		return u
	} else if host := r.Header.Get("Host"); host != "" {
		u, _ := url.Parse(host)
		return u
	}
	return fallback
}
