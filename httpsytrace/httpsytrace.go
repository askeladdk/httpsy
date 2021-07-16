// Package httpsytrace provides an interface to hook into calls made
// to ResponseWriter. It can be used to capture any HTTP response metrics,
// such as response time, bytes written and status code from your
// application's middleware.
// It can also be used to implement on-the-fly compression, hashing,
// or anything else you can think of.
//
// Package httpsytrace is the server-side equivalent of net/http/httptrace.
package httpsytrace

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
)

// ServerTracer exposes hooks into the ResponseWriter.
type ServerTracer interface {
	// Header is called when ResponseWriter.Header() is called.
	Header(http.ResponseWriter) http.Header

	// Write is called whenever the ResponseWriter is written to,
	// unless the source Reader is an *os.File and the underlying
	// TCP connection implements the io.ReaderFrom fast path.
	//
	// Wrap the *os.File in the HTTP handler to bypass
	// the fast path and intercept the writes:
	//  io.Copy(w, struct{ io.Reader }{f})
	Write(w io.Writer, p []byte) (int, error)

	// WriteHeader is called once when the status line and headers are written.
	WriteHeader(w http.ResponseWriter, statusCode int)

	// Flush is called when the http.Flusher interface is invoked.
	Flush(flusher http.Flusher)

	// Hijack is called when the http.Hijacker interface is invoked.
	Hijack(hijacker http.Hijacker) (net.Conn, *bufio.ReadWriter, error)

	// Push is called when the http.Pusher interface is invoked.
	Push(pusher http.Pusher, target string, opts *http.PushOptions) error
}

type responseWriterTracer struct {
	http.ResponseWriter
	tracer      ServerTracer
	wroteHeader int32
}

func (w *responseWriterTracer) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseWriterTracer) Header() http.Header {
	return w.tracer.Header(w.ResponseWriter)
}

func (w *responseWriterTracer) WriteHeader(statusCode int) {
	if atomic.CompareAndSwapInt32(&w.wroteHeader, 0, 1) {
		w.tracer.WriteHeader(w.ResponseWriter, statusCode)
	}
}

func (w *responseWriterTracer) Write(p []byte) (int, error) {
	w.WriteHeader(http.StatusOK)
	return w.tracer.Write(w.ResponseWriter, p)
}

func (w *responseWriterTracer) Flush() {
	w.WriteHeader(http.StatusOK)
	w.tracer.Flush(w.ResponseWriter.(http.Flusher))
}

func (w *responseWriterTracer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.tracer.Hijack(w.ResponseWriter.(http.Hijacker))
}

func (w *responseWriterTracer) Push(target string, opts *http.PushOptions) error {
	return w.tracer.Push(w.ResponseWriter.(http.Pusher), target, opts)
}

func (w *responseWriterTracer) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify() //nolint
}

func srcIsRegularFile(src io.Reader) (isRegular bool, err error) {
	// copied from the go source code:
	// https://golang.org/src/net/http/server.go?s=3003:5866#L564
	switch v := src.(type) {
	case *os.File:
		fi, err := v.Stat()
		if err != nil {
			return false, err
		}
		return fi.Mode().IsRegular(), nil
	case *io.LimitedReader:
		return srcIsRegularFile(v.R)
	default:
		return
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

var byteSlicePool = &sync.Pool{
	New: func() interface{} { return new([]byte) },
}

func (w *responseWriterTracer) ReadFrom(r io.Reader) (int64, error) {
	regular, err := srcIsRegularFile(r)
	if err != nil {
		return 0, err
	}

	w.WriteHeader(http.StatusOK)

	// fast path for regular files
	if regular {
		return w.ResponseWriter.(io.ReaderFrom).ReadFrom(r)
	}

	wf := writerFunc(func(p []byte) (int, error) { return w.tracer.Write(w.ResponseWriter, p) })
	if writerTo, ok := r.(io.WriterTo); ok {
		return writerTo.WriteTo(wf)
	}
	buf := byteSlicePool.Get().(*[]byte)
	defer byteSlicePool.Put(buf)
	return io.CopyBuffer(wf, struct{ io.Reader }{r}, *buf)
}

// Wrap hooks the ServerTracer into the ResponseWriter.
// Any calls to the ResponseWriter or its optional interfaces
// CloseNotifier, Flusher, Hijacker, Pusher, and ReaderFrom
// will be intercepted.
//
// CloseNotifier is not exposed because it is deprecated.
// ReaderFrom is not exposed because transparently calls ServerTracer.Write.
func Wrap(w http.ResponseWriter, tracer ServerTracer) http.ResponseWriter {
	const (
		ifaceCloseNotifier = 1 << iota
		ifaceFlusher
		ifaceHijacker
		ifacePusher
		ifaceReaderFrom
	)

	var ifaces int

	rw := &responseWriterTracer{w, tracer, 0}

	if _, ok := w.(http.CloseNotifier); ok { //nolint
		ifaces |= ifaceCloseNotifier // 00001
	}
	if _, ok := w.(http.Flusher); ok {
		ifaces |= ifaceFlusher // 00010
	}
	if _, ok := w.(http.Hijacker); ok {
		ifaces |= ifaceHijacker // 00100
	}
	if _, ok := w.(http.Pusher); ok {
		ifaces |= ifacePusher // 01000
	}
	if _, ok := w.(io.ReaderFrom); ok {
		ifaces |= ifaceReaderFrom // 10000
	}

	switch ifaces {
	default:
		return rw
	case ifaceCloseNotifier: // 00001
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
		}{rw, rw, rw}
	case ifaceFlusher: // 00010
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
		}{rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher: // 00011
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
		}{rw, rw, rw, rw}
	case ifaceHijacker: // 00100
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Hijacker
		}{rw, rw, rw}
	case ifaceCloseNotifier + ifaceHijacker: // 00101
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Hijacker
		}{rw, rw, rw, rw}
	case ifaceFlusher + ifaceHijacker: // 00110
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Hijacker
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker: // 00111
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Hijacker
		}{rw, rw, rw, rw, rw}
	case ifacePusher: // 01000
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Pusher
		}{rw, rw, rw}
	case ifaceCloseNotifier + ifacePusher: // 01001
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
		}{rw, rw, rw, rw}
	case ifaceFlusher + ifacePusher: // 01010
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Pusher
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifacePusher: // 01011
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Pusher
		}{rw, rw, rw, rw, rw}
	case ifaceHijacker + ifacePusher: // 01100
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Hijacker
			http.Pusher
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceHijacker + ifacePusher: // 01101
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Hijacker
			http.Pusher
		}{rw, rw, rw, rw, rw}
	case ifaceFlusher + ifaceHijacker + ifacePusher: // 01110
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
		}{rw, rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifacePusher: // 01111
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			http.Pusher
		}{rw, rw, rw, rw, rw, rw}
	case ifaceReaderFrom: // 10000
		return struct {
			Unwrapper
			http.ResponseWriter
			io.ReaderFrom
		}{rw, rw, rw}
	case ifaceCloseNotifier + ifaceReaderFrom: // 10001
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			io.ReaderFrom
		}{rw, rw, rw, rw}
	case ifaceFlusher + ifaceReaderFrom: // 10010
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			io.ReaderFrom
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifaceReaderFrom: // 10011
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceHijacker + ifaceReaderFrom: // 10100
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceHijacker + ifaceReaderFrom: // 10101
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceFlusher + ifaceHijacker + ifaceReaderFrom: // 10110
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifaceReaderFrom: // 10111
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rw, rw, rw, rw, rw, rw}
	case ifacePusher + ifaceReaderFrom: // 11000
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw}
	case ifaceCloseNotifier + ifacePusher + ifaceReaderFrom: // 11001
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceFlusher + ifacePusher + ifaceReaderFrom: // 11010
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifacePusher + ifaceReaderFrom: // 11011
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw, rw}
	case ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11100
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11101
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw, rw}
	case ifaceFlusher + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11110
		return struct {
			Unwrapper
			http.ResponseWriter
			http.Flusher
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw, rw}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11111
		return struct {
			Unwrapper
			http.ResponseWriter
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rw, rw, rw, rw, rw, rw, rw}
	}
}

// Unwrapper unwraps an underlying http.ResponseWriter.
type Unwrapper interface {
	Unwrap() http.ResponseWriter
}

// Unwrap unwraps an http.ResponseWriter that implements the Unwrapper interface.
// Use it to access possible additional interfaces that are not covered by this package.
func Unwrap(w http.ResponseWriter) (http.ResponseWriter, bool) {
	if x, ok := w.(Unwrapper); ok {
		return x.Unwrap(), true
	}
	return w, false
}

// DefaultTrace is the default implementation of ServerTracer.
type DefaultTrace struct{}

// Header implements ServerTracer.
func (st DefaultTrace) Header(w http.ResponseWriter) http.Header {
	return w.Header()
}

// WriteHeader implements ServerTracer.
func (st DefaultTrace) WriteHeader(w http.ResponseWriter, statusCode int) {
	w.WriteHeader(statusCode)
}

// Write implements ServerTracer.
func (st DefaultTrace) Write(w io.Writer, p []byte) (int, error) {
	return w.Write(p)
}

// Flush implements ServerTracer.
func (st DefaultTrace) Flush(flusher http.Flusher) {
	flusher.Flush()
}

// Hijack implements ServerTracer.
func (st DefaultTrace) Hijack(hijacker http.Hijacker) (net.Conn, *bufio.ReadWriter, error) {
	return hijacker.Hijack()
}

// Push implements ServerTracer.
func (st DefaultTrace) Push(pusher http.Pusher, target string, opts *http.PushOptions) error {
	return pusher.Push(target, opts)
}
