package gear

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// All const values got from https://github.com/labstack/echo
// MIME types
const (
	charsetUTF8 = "charset=utf-8"

	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
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

	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

type contextKey struct {
	name string
}

var nilByte []byte

// Gear global values
var (
	GearParamsKey = &contextKey{"Gear-Params-Key"}
	GearLogsKey   = &contextKey{"Gear-Logs-Key"}
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
	Param(string) string

	// Query returns the query param for the provided name.
	Query(string) string

	// Cookie returns the named cookie provided in the request.
	Cookie(string) (*http.Cookie, error)

	// SetCookie adds a `Set-Cookie` header in HTTP response.
	SetCookie(*http.Cookie)

	// Cookies returns the HTTP cookies sent with the request.
	Cookies() []*http.Cookie

	// Get retrieves data from the request Header.
	Get(string) string

	// Set saves data to the response Header.
	Set(string, string)

	// Status set a status code to response
	Status(int)

	// Type set a content type to response
	Type(string)

	// Body set a string to response.
	Body(string)

	// Error set a error message with status code to response.
	Error(*HTTPError)

	// HTML set an Html body with status code to response.
	// It will end the ctx.
	HTML(int, string) error

	// JSON set a JSON body with status code to response.
	// It will end the ctx.
	JSON(int, interface{}) error

	// JSONBlob set a JSON blob body with status code to response.
	// It will end the ctx.
	JSONBlob(int, []byte) error

	// JSONP sends a JSONP response with status code. It uses `callback` to construct the JSONP payload.
	// It will end the ctx.
	JSONP(int, string, interface{}) error

	// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
	// to construct the JSONP payload.
	// It will end the ctx.
	JSONPBlob(int, string, []byte) error

	// XML set an XML body with status code to response.
	// It will end the ctx.
	XML(int, interface{}) error

	// XMLBlob set a XML blob body with status code to response.
	// It will end the ctx.
	XMLBlob(int, []byte) error

	// Render renders a template with data and sends a text/html response with status
	// code. Templates can be registered using `app.SetRenderer()`.
	// It will end the ctx.
	Render(int, string, interface{}) error

	// Stream sends a streaming response with status code and content type.
	// It will end the ctx.
	Stream(int, string, io.Reader) error

	// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
	// client to save the file.
	// It will end the ctx.
	Attachment(string, io.ReadSeeker) error

	// Inline sends a response from `io.ReaderSeeker` as inline, opening
	// the file in the browser.
	// It will end the ctx.
	Inline(string, io.ReadSeeker) error

	// Redirect redirects the request with status code.
	// It will end the ctx.
	Redirect(int, string) error

	// End end the ctx with string body and status code optionally.
	// After it's called, the rest of middleware handles will not run.
	// But the registered hook on the ctx will run.
	End(int, []byte)

	// IsEnded return the ctx' ended status.
	IsEnded() bool

	// After add a Hook to the ctx that will run after app's Middleware.
	After(hook Hook)

	// OnEnd add a Hook to the ctx that will run before response.WriteHeader.
	OnEnd(hook Hook)

	// String returns a string represent the ctx.
	String() string
}

type gearCtx struct {
	ctx        context.Context
	cancelCtx  context.CancelFunc
	req        *http.Request
	res        *Response
	app        *Gear
	host       string
	method     string
	path       string
	ended      bool
	query      url.Values
	vals       map[interface{}]interface{}
	afterHooks []Hook
	endHooks   []Hook
	mu         sync.Mutex
}

func (ctx *gearCtx) reset(w http.ResponseWriter, req *http.Request) {
	ctx.req = req
	ctx.res.reset(w)
	ctx.ended = false
	if w == nil {
		ctx.ctx = nil
		ctx.vals = nil
		ctx.query = nil
		ctx.afterHooks = nil
		ctx.endHooks = nil
		ctx.cancelCtx = nil
	} else {
		ctx.host = req.Host
		ctx.method = req.Method
		ctx.path = normalizePath(req.URL.Path) // fix "/abc//ef" to "/abc/ef"
		ctx.vals = make(map[interface{}]interface{})
		ctx.ctx, ctx.cancelCtx = context.WithCancel(req.Context())
	}
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
	ctx.vals[key] = val
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

func (ctx *gearCtx) Param(key string) (val string) {
	if params := ctx.Value(GearParamsKey); params != nil {
		val, _ = params.(map[string]string)[key]
	}
	return
}

func (ctx *gearCtx) Query(name string) string {
	if ctx.query == nil {
		ctx.query = ctx.req.URL.Query()
	}
	return ctx.query.Get(name)
}

func (ctx *gearCtx) Cookie(name string) (*http.Cookie, error) {
	return ctx.req.Cookie(name)
}

func (ctx *gearCtx) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.res.res, cookie)
}

func (ctx *gearCtx) Cookies() []*http.Cookie {
	return ctx.req.Cookies()
}

func (ctx *gearCtx) Get(key string) string {
	return ctx.req.Header.Get(key)
}

func (ctx *gearCtx) Set(key, value string) {
	ctx.res.Set(key, value)
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
		str = MIMEApplicationJSONCharsetUTF8
	case "js":
		str = MIMEApplicationJavaScriptCharsetUTF8
	case "xml":
		str = MIMEApplicationXMLCharsetUTF8
	case "text":
		str = MIMETextPlainCharsetUTF8
	case "html":
		str = MIMETextHTMLCharsetUTF8
	}
	if str == "" {
		ctx.res.Del(HeaderContentType)
	} else {
		ctx.res.Set(HeaderContentType, str)
	}
}

func (ctx *gearCtx) Body(str string) {
	ctx.res.Body = stringToBytes(str)
}

func (ctx *gearCtx) Error(err *HTTPError) {
	ctx.End(err.Code, stringToBytes(err.Error()))
}

func (ctx *gearCtx) HTML(code int, str string) error {
	ctx.Type("html")
	ctx.End(code, stringToBytes(str))
	return nil
}

func (ctx *gearCtx) JSON(code int, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.JSONBlob(code, buf)
}

func (ctx *gearCtx) JSONBlob(code int, buf []byte) error {
	ctx.Type("json")
	ctx.End(code, buf)
	return nil
}

func (ctx *gearCtx) JSONP(code int, callback string, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.JSONPBlob(code, callback, buf)
}

func (ctx *gearCtx) JSONPBlob(code int, callback string, buf []byte) error {
	ctx.Type("js")
	buf = bytes.Join([][]byte{[]byte(callback + "("), buf, []byte(");")}, []byte{})
	ctx.End(code, buf)
	return nil
}

func (ctx *gearCtx) XML(code int, val interface{}) error {
	buf, err := xml.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.XMLBlob(code, buf)
}

func (ctx *gearCtx) XMLBlob(code int, buf []byte) error {
	ctx.Type("xml")
	ctx.End(code, buf)
	return nil
}

func (ctx *gearCtx) Render(code int, name string, data interface{}) (err error) {
	if ctx.app.Renderer == nil {
		return errors.New("renderer not registered")
	}
	buf := new(bytes.Buffer)
	if err = ctx.app.Renderer.Render(ctx, buf, name, data); err != nil {
		return
	}
	ctx.Type("html")
	ctx.End(code, buf.Bytes())
	return
}

func (ctx *gearCtx) Stream(code int, contentType string, r io.Reader) (err error) {
	ctx.End(code, nilByte)
	ctx.Type(contentType)
	_, err = io.Copy(ctx.res, r)
	return
}

func (ctx *gearCtx) Attachment(name string, content io.ReadSeeker) error {
	return ctx.contentDisposition("attachment", name, content)
}

func (ctx *gearCtx) Inline(name string, content io.ReadSeeker) error {
	return ctx.contentDisposition("inline", name, content)
}

func (ctx *gearCtx) contentDisposition(dispositionType, name string, content io.ReadSeeker) error {
	ctx.ended = true
	ctx.Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, name))
	http.ServeContent(ctx.res, ctx.req, name, time.Time{}, content)
	return nil
}

func (ctx *gearCtx) Redirect(code int, url string) error {
	ctx.ended = true
	http.Redirect(ctx.res, ctx.req, url, code)
	return nil
}

func (ctx *gearCtx) End(code int, buf []byte) {
	ctx.ended = true
	if code != 0 {
		ctx.Status(code)
	}
	if buf != nil {
		ctx.res.Body = buf
	}
}

func (ctx *gearCtx) IsEnded() bool {
	return ctx.ended || ctx.res.finished
}

func (ctx *gearCtx) After(hook Hook) {
	if !ctx.ended { // should not add afterHooks if ctx.ended
		ctx.afterHooks = append(ctx.afterHooks, hook)
	}
}

func (ctx *gearCtx) OnEnd(hook Hook) {
	if !ctx.res.finished { // should not add endHooks if ctx.res.finished
		ctx.endHooks = append(ctx.endHooks, hook)
	}
}

func (ctx *gearCtx) runAfterHooks() {
	ctx.ended = true // ensure ctx.ended to true
	for _, hook := range ctx.afterHooks {
		if ctx.res.finished {
			break
		}
		hook(ctx)
	}
	ctx.afterHooks = nil
}

func (ctx *gearCtx) runEndHooks() {
	ctx.res.finished = true // ensure ctx.res.finished to true
	for _, hook := range ctx.endHooks {
		if ctx.res.finished {
			break
		}
		hook(ctx)
	}
	ctx.endHooks = nil
}
