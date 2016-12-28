package gear

import (
	"net/http"
	"regexp"
	"strconv"
)

var defaultErrorHeaderReg = regexp.MustCompile(
	`(?i)^(accept|allow|retry-after|warning|access-control-allow-)`)

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	ctx         *Context
	res         http.ResponseWriter
	header      http.Header
	wroteHeader atomicBool
	responded   atomicBool
	bodyLength  int // number of bytes to write
	status      int // response Status Code
}

func newResponse(ctx *Context, w http.ResponseWriter) *Response {
	return &Response{ctx: ctx, res: w, header: w.Header()}
}

// Del deletes the values associated with key.
func (r *Response) Del(key string) {
	r.header.Del(key)
}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (r *Response) Get(key string) string {
	return r.header.Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (r *Response) Set(key, value string) {
	r.header.Set(key, value)
}

// Status sets status code to the response
func (r *Response) Status(status int) {
	r.status = status
}

// GetStatus returns status code of the response
func (r *Response) GetStatus() int {
	return r.status
}

// ResetHeader reset headers. If keepSubset is true,
// header matching `(?i)^(accept|allow|retry-after|warning|access-control-allow-)` will be keep
func (r *Response) ResetHeader(regs ...*regexp.Regexp) {
	reg := defaultErrorHeaderReg
	if len(regs) > 0 {
		reg = regs[0]
	}
	for key := range r.header {
		if !reg.MatchString(key) {
			delete(r.header, key)
		}
	}
}

// Header returns the header map that will be sent by WriteHeader.
func (r *Response) Header() http.Header {
	return r.header
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(buf []byte) (int, error) {
	// Some http Handler will call Write directly.
	if !r.wroteHeader.isTrue() {
		r.WriteHeader(0)
	}
	return r.res.Write(buf)
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

	// check Body and Content Length
	if r.bodyLength > 0 && r.header.Get(HeaderContentLength) == "" {
		r.header.Set(HeaderContentLength, strconv.Itoa(r.bodyLength))
	}

	// check status, r.status maybe changed in afterHooks
	if r.status <= 0 {
		if r.bodyLength > 0 {
			r.status = http.StatusOK
		} else {
			r.status = 444 // 444 No Response (from Nginx)
		}
	}
	r.res.WriteHeader(r.status)
	// execute "end hooks" in LIFO order after Response.WriteHeader
	for i := len(r.ctx.endHooks) - 1; i >= 0; i-- {
		r.ctx.endHooks[i]()
	}
}

func (r *Response) respond(status int, body []byte) (err error) {
	if r.responded.swapTrue() && !r.wroteHeader.isTrue() {
		r.bodyLength = len(body)
		r.WriteHeader(status)
		if r.bodyLength > 0 {
			_, err = r.Write(body)
		}
	}
	return
}

// HeaderWrote indecates that whether the reply header has been (logically) written.
func (r *Response) HeaderWrote() bool {
	return r.wroteHeader.isTrue()
}
