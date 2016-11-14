package gear

import (
	"net/http"
	"regexp"
	"strconv"
)

var defaultErrorHeaderReg = regexp.MustCompile(`(?i)^(accept|allow|retry-after|warning|access-control-allow-)`)

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	ctx         *Context
	res         http.ResponseWriter
	header      http.Header
	wroteHeader bool

	Body   []byte // response Content
	Status int    // response Status Code
	Type   string // response Content-Type
}

func newResponse(ctx *Context, w http.ResponseWriter) *Response {
	return &Response{ctx: ctx, res: w, header: w.Header()}
}

// Add adds the key, value pair to the header. It appends to any existing values associated with key.
func (r *Response) Add(key, value string) {
	r.header.Add(key, value)
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

// ResetHeader reset headers. If keepSubset is true,
// header matching `(?i)^(accept|allow|retry-after|warning|access-control-allow-)` will be keep
func (r *Response) ResetHeader(keepSubset bool) {
	for key := range r.header {
		if !keepSubset || !defaultErrorHeaderReg.MatchString(key) {
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
	if !r.wroteHeader {
		r.WriteHeader(r.Status)
	}
	return r.res.Write(buf)
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to Write
// will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to send error codes.
func (r *Response) WriteHeader(code int) {
	r.ctx.mu.Lock()
	if r.wroteHeader {
		r.ctx.mu.Unlock()
		return
	}
	r.wroteHeader = true
	r.ctx.mu.Unlock()

	r.Status = code
	// ensure that ended is true
	r.ctx.setEnd(false)
	// execute "after hooks" in LIFO order before Response.WriteHeader
	for i := len(r.ctx.afterHooks) - 1; i >= 0; i-- {
		r.ctx.afterHooks[i]()
	}
	r.ctx.afterHooks = nil
	// r.Status maybe changed in hooks
	// check Status
	if r.Status <= 0 {
		if r.Body != nil {
			r.Status = http.StatusOK
		} else {
			r.Status = 444 // 444 No Response (from Nginx)
		}
	}
	// check Body and Content Length
	if r.Body != nil && r.header.Get(HeaderContentLength) == "" {
		r.header.Set(HeaderContentLength, strconv.FormatInt(int64(len(r.Body)), 10))
	}
	r.res.WriteHeader(r.Status)
	// execute "end hooks" in LIFO order after Response.WriteHeader
	for i := len(r.ctx.endHooks) - 1; i >= 0; i-- {
		r.ctx.endHooks[i]()
	}
	r.ctx.endHooks = nil
}

// HeaderWrote indecates that whether the reply header has been (logically) written.
func (r *Response) HeaderWrote() bool {
	return r.wroteHeader
}

func (r *Response) respond() (err error) {
	if !r.wroteHeader {
		r.WriteHeader(r.Status)
		if r.Body != nil {
			_, err = r.Write(r.Body)
		}
	}
	return
}
