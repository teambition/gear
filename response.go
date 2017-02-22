package gear

import (
	"bufio"
	"net"
	"net/http"
	"regexp"
	"strconv"
)

var defaultHeaderFilterReg = regexp.MustCompile(
	`(?i)^(accept|allow|retry-after|warning|vary|access-control-allow-)`)

// ErrPusherNotImplemented is return from Response.Push.
var ErrPusherNotImplemented = NewAppError("http.Pusher not implemented")

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	ctx         *Context
	w           http.ResponseWriter // the origin http.ResponseWriter, should not be override.
	rw          http.ResponseWriter // maybe a http.ResponseWriter wrapper
	wroteHeader atomicBool
	responded   atomicBool
	bodyLength  int // number of bytes to write, ignore stream body.
	status      int // response Status Code
}

func newResponse(ctx *Context, w http.ResponseWriter) *Response {
	return &Response{ctx: ctx, w: w, rw: w}
}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (r *Response) Get(key string) string {
	return r.Header().Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (r *Response) Set(key, value string) {
	r.Header().Set(key, value)
}

// Del deletes the header entries associated with key.
func (r *Response) Del(key string) {
	r.Header().Del(key)
}

// Vary manipulate the HTTP Vary header.
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Vary
func (r *Response) Vary(field string) {
	if field != "" && r.Get(HeaderVary) != "*" {
		if field == "*" {
			r.Header().Set(HeaderVary, field)
		} else {
			r.Header().Add(HeaderVary, field)
		}
	}
}

// ResetHeader reset headers. If keepSubset is true,
// header matching `(?i)^(accept|allow|retry-after|warning|access-control-allow-)` will be keep
func (r *Response) ResetHeader(filterReg ...*regexp.Regexp) {
	reg := defaultHeaderFilterReg
	if len(filterReg) > 0 {
		reg = filterReg[0]
	}
	header := r.Header()
	for key := range header {
		if !reg.MatchString(key) {
			delete(header, key)
		}
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (r *Response) Header() http.Header {
	return r.rw.Header()
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(buf []byte) (int, error) {
	// Some http Handler will call Write directly.
	if !r.wroteHeader.isTrue() {
		if r.status == 0 {
			r.status = 200
		}
		r.WriteHeader(0)
	}
	return r.rw.Write(buf)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to send error codes.
func (r *Response) WriteHeader(code int) {
	if !r.wroteHeader.swapTrue() {
		return
	}
	// ensure that ended is true
	r.ctx.ended.setTrue()

	// set status before afterHooks
	if code > 0 {
		r.status = code
	}

	// execute "after hooks" in LIFO order before Response.WriteHeader
	for i := len(r.ctx.afterHooks) - 1; i >= 0; i-- {
		r.ctx.afterHooks[i]()
	}

	// check status, r.status maybe changed in afterHooks
	if !IsStatusCode(r.status) {
		if r.bodyLength > 0 {
			r.status = http.StatusOK
		} else {
			// Misdirected request, http://tools.ietf.org/html/rfc7540#section-9.1.2
			// The request was directed at a server that is not able to produce a response.
			r.status = 421
		}
	} else if isEmptyStatus(r.status) {
		r.bodyLength = 0
	}

	// check and set Content-Length
	if r.bodyLength > 0 && r.Get(HeaderContentLength) == "" {
		r.Set(HeaderContentLength, strconv.Itoa(r.bodyLength))
	}
	r.rw.WriteHeader(r.status)
	// execute "end hooks" in LIFO order after Response.WriteHeader
	for i := len(r.ctx.endHooks) - 1; i >= 0; i-- {
		r.ctx.endHooks[i]()
	}
}

// Flush implements the http.Flusher interface to allow an HTTP handler to flush
// buffered data to the client.
// See [http.Flusher](https://golang.org/pkg/net/http/#Flusher)
func (r *Response) Flush() {
	r.w.(http.Flusher).Flush()
}

// Hijack implements the http.Hijacker interface to allow an HTTP handler to
// take over the connection.
// See [http.Hijacker](https://golang.org/pkg/net/http/#Hijacker)
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.w.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotifier interface to allow detecting
// when the underlying connection has gone away.
// This mechanism can be used to cancel long operations on the server if the
// client has disconnected before the response is ready.
// See [http.CloseNotifier](https://golang.org/pkg/net/http/#CloseNotifier)
func (r *Response) CloseNotify() <-chan bool {
	return r.w.(http.CloseNotifier).CloseNotify()
}

// Push implements http.Pusher.
// Example: https://github.com/teambition/gear/blob/master/example/http2/app.go
func (r *Response) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return ErrPusherNotImplemented
}

// HeaderWrote indecates that whether the reply header has been (logically) written.
func (r *Response) HeaderWrote() bool {
	return r.wroteHeader.isTrue()
}

func (r *Response) respond(status int, body []byte) (err error) {
	if r.responded.swapTrue() && !r.wroteHeader.isTrue() {
		r.bodyLength = len(body)
		r.WriteHeader(status)
		// bodyLength will reset to 0 with empty status
		if r.bodyLength > 0 {
			_, err = r.Write(body)
		}
	}
	return
}

// IsStatusCode returns true if status is HTTP status code.
// https://en.wikipedia.org/wiki/List_of_HTTP_status_codes
func IsStatusCode(status int) bool {
	switch status {
	case 100, 101, 102,
		200, 201, 202, 203, 204, 205, 206, 207, 208, 226,
		300, 301, 302, 303, 304, 305, 306, 307, 308,
		400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 418,
		421, 422, 423, 424, 426, 428, 429, 431, 440, 444, 449, 450, 451, 494, 495, 496, 497, 498, 499,
		500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511, 520, 521, 522, 523, 524, 525, 526, 527:
		return true
	default:
		return false
	}
}

func isRedirectStatus(status int) bool {
	switch status {
	case 300, 301, 302, 303, 305, 307, 308:
		return true
	default:
		return false
	}
}

func isEmptyStatus(status int) bool {
	switch status {
	case 204, 205, 304:
		return true
	default:
		return false
	}
}
