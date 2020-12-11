// Package httpsytrace provides an interface to hook into calls made
// to ResponseWriter. It can be used to capture HTTP response metrics
// such as response time, bytes written and status code from your
// application's middleware.
// It is the server-side equivalent of net/http/httptrace.
package httpsytrace

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	ifaceCloseNotifier = 1 << iota
	ifaceFlusher
	ifaceHijacker
	ifacePusher
	ifaceReaderFrom
)

// Tracer exposes hooks into the ResponseWriter.
type Tracer interface {
	// Write is called whenever the response body is written to,
	// either directly by Write or via the io.ReaderFrom interface.
	Write(w io.Writer, p []byte) (int, error)

	// WriteHeader is called once when the status line and headers are written.
	WriteHeader(w http.ResponseWriter, statusCode int)

	// Flush is called when the http.Flusher interface is invoked.
	Flush(flusher http.Flusher)

	// Hijack is called when the Hijacker interface is invoked.
	Hijack(hijacker http.Hijacker) (net.Conn, *bufio.ReadWriter, error)

	// Push is called when the Pusher interface is invoked.
	Push(pusher http.Pusher, target string, opts *http.PushOptions) error
}

type responseWriterTracer struct {
	http.ResponseWriter
	tracer      Tracer
	wroteHeader int32
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

func (w *responseWriterTracer) flush() {
	w.WriteHeader(http.StatusOK)
	w.tracer.Flush(w.ResponseWriter.(http.Flusher))
}

func (w *responseWriterTracer) hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.tracer.Hijack(w.ResponseWriter.(http.Hijacker))
}

func (w *responseWriterTracer) push(target string, opts *http.PushOptions) error {
	return w.tracer.Push(w.ResponseWriter.(http.Pusher), target, opts)
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}

func (w *responseWriterTracer) readFrom(r io.Reader) (int64, error) {
	w.WriteHeader(http.StatusOK)
	pr, pw := io.Pipe()
	defer pr.Close()

	traceWriter := writerFunc(func(p []byte) (int, error) {
		return w.tracer.Write(pw, p)
	})

	go func() {
		_, err := io.Copy(traceWriter, r)
		_ = pw.CloseWithError(err)
	}()

	return w.ResponseWriter.(io.ReaderFrom).ReadFrom(pr)
}

type (
	flusherProxy    struct{ w *responseWriterTracer }
	hijackerProxy   struct{ w *responseWriterTracer }
	pusherProxy     struct{ w *responseWriterTracer }
	readerFromProxy struct{ w *responseWriterTracer }
)

func (t flusherProxy) Flush() {
	t.w.flush()
}

func (t hijackerProxy) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return t.w.hijack()
}

func (t pusherProxy) Push(target string, opts *http.PushOptions) error {
	return t.w.push(target, opts)
}

func (t readerFromProxy) ReadFrom(r io.Reader) (int64, error) {
	return t.w.readFrom(r)
}

// Hook hooks the Tracer into the ResponseWriter.
// Any calls to the ResponseWriter or its optional interfaces
// CloseNotifier, Flusher, Hijacker, Pusher, and ReaderFrom
// will go through the Tracer.
//
// CloseNotifier and ReaderFrom are not exposed directly.
// CloseNotifier is deprecated and is not useful to hook into.
// ReaderFrom transparently writes to the Tracer and does not need to be exposed.
func Hook(w http.ResponseWriter, tracer Tracer) http.ResponseWriter {
	var (
		closeNotifier http.CloseNotifier // 00001
		flusher       http.Flusher       // 00010
		hijacker      http.Hijacker      // 00100
		pusher        http.Pusher        // 01000
		readerFrom    io.ReaderFrom      // 10000
		ifaces        int
		ok            bool
	)

	rwt := &responseWriterTracer{w, tracer, 0}

	if closeNotifier, ok = w.(http.CloseNotifier); ok {
		ifaces |= ifaceCloseNotifier
	}
	if _, ok = w.(http.Flusher); ok {
		ifaces |= ifaceFlusher
		flusher = flusherProxy{rwt}
	}
	if _, ok = w.(http.Hijacker); ok {
		ifaces |= ifaceHijacker
		hijacker = hijackerProxy{rwt}
	}
	if _, ok = w.(http.Pusher); ok {
		ifaces |= ifacePusher
		pusher = pusherProxy{rwt}
	}
	if _, ok = w.(io.ReaderFrom); ok {
		ifaces |= ifaceReaderFrom
		readerFrom = readerFromProxy{rwt}
	}

	switch ifaces {
	default:
		return rwt
	case ifaceCloseNotifier: // 00001
		return struct {
			*responseWriterTracer
			http.CloseNotifier
		}{rwt, closeNotifier}
	case ifaceFlusher: // 00010
		return struct {
			*responseWriterTracer
			http.Flusher
		}{rwt, flusher}
	case ifaceCloseNotifier + ifaceFlusher: // 00011
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
		}{rwt, closeNotifier, flusher}
	case ifaceHijacker: // 00100
		return struct {
			*responseWriterTracer
			http.Hijacker
		}{rwt, hijacker}
	case ifaceCloseNotifier + ifaceHijacker: // 00101
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Hijacker
		}{rwt, closeNotifier, hijacker}
	case ifaceFlusher + ifaceHijacker: // 00110
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Hijacker
		}{rwt, flusher, hijacker}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker: // 00111
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Hijacker
		}{rwt, closeNotifier, flusher, hijacker}
	case ifacePusher: // 01000
		return struct {
			*responseWriterTracer
			http.Pusher
		}{rwt, pusher}
	case ifaceCloseNotifier + ifacePusher: // 01001
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Pusher
		}{rwt, closeNotifier, pusher}
	case ifaceFlusher + ifacePusher: // 01010
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Pusher
		}{rwt, flusher, pusher}
	case ifaceCloseNotifier + ifaceFlusher + ifacePusher: // 01011
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Pusher
		}{rwt, closeNotifier, flusher, pusher}
	case ifaceHijacker + ifacePusher: // 01100
		return struct {
			*responseWriterTracer
			http.Hijacker
			http.Pusher
		}{rwt, hijacker, pusher}
	case ifaceCloseNotifier + ifaceHijacker + ifacePusher: // 01101
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Hijacker
			http.Pusher
		}{rwt, closeNotifier, hijacker, pusher}
	case ifaceFlusher + ifaceHijacker + ifacePusher: // 01110
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Hijacker
			http.Pusher
		}{rwt, flusher, hijacker, pusher}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifacePusher: // 01111
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			http.Pusher
		}{rwt, closeNotifier, flusher, hijacker, pusher}
	case ifaceReaderFrom: // 10000
		return struct {
			*responseWriterTracer
			io.ReaderFrom
		}{rwt, readerFrom}
	case ifaceCloseNotifier + ifaceReaderFrom: // 10001
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			io.ReaderFrom
		}{rwt, closeNotifier, readerFrom}
	case ifaceFlusher + ifaceReaderFrom: // 10010
		return struct {
			*responseWriterTracer
			http.Flusher
			io.ReaderFrom
		}{rwt, flusher, readerFrom}
	case ifaceCloseNotifier + ifaceFlusher + ifaceReaderFrom: // 10011
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			io.ReaderFrom
		}{rwt, closeNotifier, flusher, readerFrom}
	case ifaceHijacker + ifaceReaderFrom: // 10100
		return struct {
			*responseWriterTracer
			http.Hijacker
			io.ReaderFrom
		}{rwt, hijacker, readerFrom}
	case ifaceCloseNotifier + ifaceHijacker + ifaceReaderFrom: // 10101
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Hijacker
			io.ReaderFrom
		}{rwt, closeNotifier, hijacker, readerFrom}
	case ifaceFlusher + ifaceHijacker + ifaceReaderFrom: // 10110
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rwt, flusher, hijacker, readerFrom}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifaceReaderFrom: // 10111
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			io.ReaderFrom
		}{rwt, closeNotifier, flusher, hijacker, readerFrom}
	case ifacePusher + ifaceReaderFrom: // 11000
		return struct {
			*responseWriterTracer
			http.Pusher
			io.ReaderFrom
		}{rwt, pusher, readerFrom}
	case ifaceCloseNotifier + ifacePusher + ifaceReaderFrom: // 11001
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Pusher
			io.ReaderFrom
		}{rwt, closeNotifier, pusher, readerFrom}
	case ifaceFlusher + ifacePusher + ifaceReaderFrom: // 11010
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Pusher
			io.ReaderFrom
		}{rwt, flusher, pusher, readerFrom}
	case ifaceCloseNotifier + ifaceFlusher + ifacePusher + ifaceReaderFrom: // 11011
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Pusher
			io.ReaderFrom
		}{rwt, closeNotifier, flusher, pusher, readerFrom}
	case ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11100
		return struct {
			*responseWriterTracer
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rwt, hijacker, pusher, readerFrom}
	case ifaceCloseNotifier + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11101
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rwt, closeNotifier, hijacker, pusher, readerFrom}
	case ifaceFlusher + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11110
		return struct {
			*responseWriterTracer
			http.Flusher
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rwt, flusher, hijacker, pusher, readerFrom}
	case ifaceCloseNotifier + ifaceFlusher + ifaceHijacker + ifacePusher + ifaceReaderFrom: // 11111
		return struct {
			*responseWriterTracer
			http.CloseNotifier
			http.Flusher
			http.Hijacker
			http.Pusher
			io.ReaderFrom
		}{rwt, closeNotifier, flusher, hijacker, pusher, readerFrom}
	}
}

// Unwrap returns the underlying ResponseWriter.
// Use it to access possible additional interfaces that are not
// covered by this package.
func Unwrap(w http.ResponseWriter) http.ResponseWriter {
	if x, ok := w.(*responseWriterTracer); ok {
		return x.ResponseWriter
	}
	return w
}

// Metrics implements Tracer and collects basic metrics.
// Its behaviour can be extended by embedding it in another struct.
type Metrics struct {
	// Header is the response header.
	Header http.Header

	// BytesWritten is the total amount of bytes written to the response.
	BytesWritten int64

	// StatusCode is the HTTP status code of the response.
	StatusCode int

	// Start is the moment when taking measurements started.
	Start time.Time
}

// WriteHeader implements Tracer.
func (t *Metrics) WriteHeader(w http.ResponseWriter, statusCode int) {
	t.Header = w.Header()
	t.StatusCode = statusCode
	w.WriteHeader(statusCode)
}

// Write implements Tracer.
func (t *Metrics) Write(w io.Writer, p []byte) (int, error) {
	t.BytesWritten += int64(len(p))
	return w.Write(p)
}

// Flush implements Tracer. It invokes flusher.
func (t Metrics) Flush(flusher http.Flusher) {
	flusher.Flush()
}

// Hijack implements Tracer. It invokes hijacker and passes its return values.
func (t Metrics) Hijack(hijacker http.Hijacker) (net.Conn, *bufio.ReadWriter, error) {
	return hijacker.Hijack()
}

// Push implements Tracer. It invokes pusher and passes its return values.
func (t Metrics) Push(pusher http.Pusher, target string, opts *http.PushOptions) error {
	return pusher.Push(target, opts)
}

// Duration is the amount of time that has elapsed since Start.
func (t Metrics) Duration() time.Duration {
	return time.Since(t.Start)
}
