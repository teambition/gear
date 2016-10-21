package gear

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// Headers
const (
	HeaderAcceptEncoding                = "Accept-Encoding"
	HeaderAllow                         = "Allow"
	HeaderAuthorization                 = "Authorization"
	HeaderContentDisposition            = "Content-Disposition"
	HeaderContentEncoding               = "Content-Encoding"
	HeaderContentLength                 = "Content-Length"
	HeaderContentType                   = "Content-Type"
	HeaderCookie                        = "Cookie"
	HeaderSetCookie                     = "Set-Cookie"
	HeaderIfModifiedSince               = "If-Modified-Since"
	HeaderLastModified                  = "Last-Modified"
	HeaderLocation                      = "Location"
	HeaderUpgrade                       = "Upgrade"
	HeaderVary                          = "Vary"
	HeaderWWWAuthenticate               = "WWW-Authenticate"
	HeaderXForwardedProto               = "X-Forwarded-Proto"
	HeaderXHTTPMethodOverride           = "X-HTTP-Method-Override"
	HeaderXForwardedFor                 = "X-Forwarded-For"
	HeaderXRealIP                       = "X-Real-IP"
	HeaderServer                        = "Server"
	HeaderOrigin                        = "Origin"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

const (
	// CharsetUTF8
	CharsetUTF8 = "charset=utf-8"

	// MediaTypes
	ApplicationJSON                  = "application/json"
	ApplicationJSONCharsetUTF8       = ApplicationJSON + "; " + CharsetUTF8
	ApplicationJavaScript            = "application/javascript"
	ApplicationJavaScriptCharsetUTF8 = ApplicationJavaScript + "; " + CharsetUTF8
	ApplicationXML                   = "application/xml"
	ApplicationXMLCharsetUTF8        = ApplicationXML + "; " + CharsetUTF8
	ApplicationForm                  = "application/x-www-form-urlencoded"
	ApplicationProtobuf              = "application/protobuf"
	TextHTML                         = "text/html"
	TextHTMLCharsetUTF8              = TextHTML + "; " + CharsetUTF8
	TextPlain                        = "text/plain"
	TextPlainCharsetUTF8             = TextPlain + "; " + CharsetUTF8
	MultipartForm                    = "multipart/form-data"

	GearValueParams = "GearValueParams"
)

// Context represents the context of the current HTTP request. It holds request and
// response objects, path, path parameters, data, registered handler and content.Context.
type Context interface {
	context.Context

	// Cancel cancel the ctx and all it' children context
	Cancel()

	// WithCancel returns a copy of the ctx with a new Done channel.
	// The returned context's Done channel is closed when the returned cancel function is called or when the parent context's Done channel is closed, whichever happens first.
	WithCancel() (context.Context, context.CancelFunc)

	// WithDeadline returns a copy of the ctx with the deadline adjusted to be no later than d.
	WithDeadline(time.Time) (context.Context, context.CancelFunc)

	// WithTimeout returns WithDeadline(time.Now().Add(timeout)).
	WithTimeout(time.Duration) (context.Context, context.CancelFunc)

	// WithValue returns a copy of the ctx in which the value associated with key is val.
	WithValue(interface{}, interface{}) context.Context

	// SetValue save a key and value to the ctx.
	SetValue(interface{}, interface{})

	// Lock locks ctx for writing. If the lock is already locked for reading or writing, Lock blocks until the lock is available.
	Lock()

	// Unlock unlocks ctx for writing. It is a run-time error if rw is not locked for writing on entry to Unlock.
	Unlock()

	// Request returns `*http.Request`.
	Request() *http.Request

	// Request returns `*Response`.
	Response() *Response

	// IP returns the client's network address based on `X-Forwarded-For`
	// or `X-Real-IP` request header.
	IP() string

	// Host returns the the client's request Host.
	Host() string

	// Method returns the client's request Method.
	Method() string

	// Path returns the client's request Path.
	Path() string

	// Param returns path parameter by name.
	Param(string) (string, bool)

	// Query returns the query param for the provided name.
	// Query(string) string TODO

	// Cookie returns the named cookie provided in the request.
	// Cookie(string) (*http.Cookie, error) TODO

	// SetCookie adds a `Set-Cookie` header in HTTP response.
	// SetCookie(*http.Cookie) TODO

	// Cookies returns the HTTP cookies sent with the request.
	// Cookies() []*http.Cookie TODO

	// Get retrieves data from the request Header.
	Get(string) string

	// Set saves data to the response Header.
	Set(string, string)

	// Body set a string to response.
	Body(string)

	// Status set a status code to response
	Status(int)

	// Type set a content type to response
	Type(string)

	// HTML set an Html body with status code to response.
	HTML(int, string)

	// JSON set a JSON body with status code to response.
	JSON(int, interface{})

	// JSONBlob set a JSON blob body with status code to response.
	// JSONBlob(int, []byte) TODO

	// XML set an XML body with status code to response.
	// XML(int, interface{}) TODO

	// XMLBlob set a XML blob body with status code to response.
	// XMLBlob(int, []byte) TODO

	// Render renders a template with data and sends a text/html response with status
	// code. Templates can be registered using `Echo.SetRenderer()`.
	// Render(int, string, interface{}) error TODO

	// Stream sends a streaming response with status code and content type.
	// Stream(int, string, io.Reader) error TODO

	// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
	// client to save the file.
	// Attachment(io.ReadSeeker, string) error TODO

	// Inline sends a response from `io.ReaderSeeker` as inline, opening
	// the file in the browser.
	// Inline(io.ReadSeeker, string) error TODO

	// Redirect redirects the request with status code.
	// Redirect(int, string) error TODO

	// End send an string body with status code to response.
	End(int, string)

	// IsEnded return the ctx' ended status.
	IsEnded() bool

	// After add a Middleware handle to the ctx that will run after app'Middleware.
	After(handle Middleware)

	// String returns a string represent the ctx.
	String() string
}

// Context docs
type gearCtx struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	req       *http.Request
	res       *Response
	host      string
	method    string
	path      string
	ended     bool
	vals      map[interface{}]interface{}
	hooks     []Middleware
	mu        sync.Mutex
}

// NewContext returns a Context
func (ctx *gearCtx) reset(w http.ResponseWriter, req *http.Request) {
	ctx.ctx, ctx.cancelCtx = context.WithCancel(req.Context())
	ctx.req = req
	ctx.res.reset(w)
	ctx.host = req.Host
	ctx.method = req.Method
	ctx.ended = false
	ctx.path = normalizePath(req.URL.Path) // convert "/abc//ef" to "/abc/ef"
	ctx.hooks = make([]Middleware, 0)
	ctx.vals = make(map[interface{}]interface{})
}

// ----- implement context.Context interface -----

func (ctx *gearCtx) Deadline() (time.Time, bool) {
	return ctx.ctx.Deadline()
}

func (ctx *gearCtx) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *gearCtx) Err() error {
	return ctx.ctx.Err()
}

func (ctx *gearCtx) Value(key interface{}) (val interface{}) {
	var ok bool
	if val, ok = ctx.vals[key]; !ok {
		val = ctx.ctx.Value(key)
	}
	return
}

func (ctx *gearCtx) String() string {
	return fmt.Sprintf("gweb.Context{Req: %v, Res: %v}", ctx.req, ctx.res)
}

func (ctx *gearCtx) Cancel() {
	ctx.cancelCtx()
}

func (ctx *gearCtx) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx.ctx)
}

func (ctx *gearCtx) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx.ctx, deadline)
}

func (ctx *gearCtx) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx.ctx, timeout)
}

func (ctx *gearCtx) WithValue(key, val interface{}) context.Context {
	return context.WithValue(ctx.ctx, key, val)
}

func (ctx *gearCtx) SetValue(key, val interface{}) {
	ctx.Lock()
	ctx.vals[key] = val
	ctx.Unlock()
}

// ----- implement Locker interface -----

func (ctx *gearCtx) Lock() {
	ctx.mu.Lock()
}

func (ctx *gearCtx) Unlock() {
	ctx.mu.Unlock()
}

func (ctx *gearCtx) Request() *http.Request {
	return ctx.req
}

func (ctx *gearCtx) Response() *Response {
	return ctx.res
}

func (ctx *gearCtx) IP() string {
	ra := ctx.req.RemoteAddr
	if ip := ctx.req.Header.Get(HeaderXForwardedFor); ip != "" {
		ra = ip
	} else if ip := ctx.req.Header.Get(HeaderXRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

func (ctx *gearCtx) Host() string {
	return ctx.host
}

func (ctx *gearCtx) Method() string {
	return ctx.method
}

func (ctx *gearCtx) Path() string {
	return ctx.path
}

func (ctx *gearCtx) Param(key string) (val string, ok bool) {
	if params := ctx.Value("GearValueParams"); params != nil {
		val, ok = params.(map[string]string)[key]
	}
	return
}

func (ctx *gearCtx) Get(key string) string {
	return ctx.req.Header.Get(key)
}

func (ctx *gearCtx) Set(key, value string) {
	ctx.res.Set(key, value)
}

func (ctx *gearCtx) Body(str string) {
	ctx.res.stringBody(str)
}

func (ctx *gearCtx) Status(code int) {
	if statusText := http.StatusText(code); statusText == "" {
		code = 500
	}
	ctx.res.Status = code
}

func (ctx *gearCtx) Type(str string) {
	switch str {
	case "json":
		str = ApplicationJSONCharsetUTF8
	case "js":
		str = ApplicationJavaScriptCharsetUTF8
	case "xml":
		str = ApplicationXMLCharsetUTF8
	case "text":
		str = TextPlainCharsetUTF8
	case "html":
		str = TextHTMLCharsetUTF8
	}
	if str == "" {
		ctx.res.Del(HeaderContentType)
	} else {
		ctx.res.Set(HeaderContentType, str)
	}
}

func (ctx *gearCtx) HTML(code int, str string) {
	ctx.Status(code)
	ctx.Type("html")
	ctx.Body(str)
}

func (ctx *gearCtx) JSON(code int, val interface{}) {
	buf, err := json.Marshal(val)

	if err != nil {
		ctx.Status(500)
		ctx.Type("text")
		ctx.Body(err.Error())
	} else {
		ctx.Status(code)
		ctx.Type("json")
		ctx.res.Body = buf
	}
}

// func (ctx *gearCtx) Attachment(filename string) {

// }

// func (ctx *gearCtx) Redirect(url string) {

// }

func (ctx *gearCtx) End(code int, str string) {
	ctx.ended = true
	if code != 0 {
		ctx.Status(code)
	}
	if str != "" {
		ctx.Body(str)
	}
}

func (ctx *gearCtx) IsEnded() bool {
	return ctx.ended || ctx.res.finished
}

func (ctx *gearCtx) After(handle Middleware) {
	ctx.hooks = append(ctx.hooks, handle)
}
