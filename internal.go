package httpsy

import (
	"bytes"
	"crypto/rand"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
)

type errorString string

func (e errorString) Error() string {
	return string(e)
}

var bytesBufferPool = &sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(nil)
	},
}

func getBytesBuffer() *bytes.Buffer {
	b := bytesBufferPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func putBytesBuffer(b *bytes.Buffer) {
	bytesBufferPool.Put(b)
}

func randomNoise(b []byte) {
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
}

func bytesToASCII(b []byte) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

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
	return url1.Scheme == url2.Scheme && url1.Host == url2.Host
}
