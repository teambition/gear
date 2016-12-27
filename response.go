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

	body   []byte // response Content
	status int    // response Status Code
}

func newResponse(ctx *Context, w http.ResponseWriter) *Response {
	return &Response{ctx: ctx, res: w, header: w.Header()}
}

// Del deletes the values associated with key.
func (r *Response) Del(key string) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	if !r.wroteHeader {
		r.header.Del(key)
	}
}

// Get gets the first value associated with the given key. If there are no values associated with the key, Get returns "". To access multiple values of a key, access the map directly with CanonicalHeaderKey.
func (r *Response) Get(key string) string {
	return r.header.Get(key)
}

// Set sets the header entries associated with key to the single element value. It replaces any existing values associated with key.
func (r *Response) Set(key, value string) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	if !r.wroteHeader {
		r.header.Set(key, value)
	}
}

// GetStatus returns status code of the response
func (r *Response) GetStatus() int {
	r.ctx.mu.RLock()
	defer r.ctx.mu.RUnlock()
	return r.status
}

// SetStatus sets status code to the response
func (r *Response) SetStatus(status int) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	r.status = status
}

// GetLen returns byte length of the response content
func (r *Response) GetLen() int {
	return len(r.body)
}

func (r *Response) setBody(body []byte) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	r.body = body
}

// ResetHeader reset headers. If keepSubset is true,
// header matching `(?i)^(accept|allow|retry-after|warning|access-control-allow-)` will be keep
func (r *Response) ResetHeader(keepSubset bool) {
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
	if !r.wroteHeader {
		for key := range r.header {
			if !keepSubset || !defaultErrorHeaderReg.MatchString(key) {
				delete(r.header, key)
			}
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
	if !r.HeaderWrote() {
		r.WriteHeader(r.status)
	}
	r.ctx.mu.Lock()
	defer r.ctx.mu.Unlock()
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
	r.status = code
	// ensure that ended is true
	r.ctx.ended = true
	r.ctx.mu.Unlock()

	// execute "after hooks" in LIFO order before Response.WriteHeader
	for i := len(r.ctx.afterHooks) - 1; i >= 0; i-- {
		r.ctx.afterHooks[i]()
	}
	r.ctx.cleanAfterHooks()
	// r.Status maybe changed in hooks
	// check Status
	if r.status <= 0 {
		if r.body != nil {
			r.status = http.StatusOK
		} else {
			r.status = 444 // 444 No Response (from Nginx)
		}
	}
	// check Body and Content Length
	if r.body != nil && r.header.Get(HeaderContentLength) == "" {
		r.header.Set(HeaderContentLength, strconv.FormatInt(int64(len(r.body)), 10))
	}
	r.res.WriteHeader(r.status)
	// execute "end hooks" in LIFO order after Response.WriteHeader
	for i := len(r.ctx.endHooks) - 1; i >= 0; i-- {
		r.ctx.endHooks[i]()
	}
	r.ctx.cleanEndHooks()
}

// HeaderWrote indecates that whether the reply header has been (logically) written.
func (r *Response) HeaderWrote() bool {
	r.ctx.mu.RLock()
	defer r.ctx.mu.RUnlock()
	return r.wroteHeader
}

func (r *Response) respond() (err error) {
	if !r.HeaderWrote() {
		r.WriteHeader(r.status)
		if r.body != nil {
			_, err = r.Write(r.body)
		}
	}
	return
}
