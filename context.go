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
	"time"
)

// MIME types
const (
	// All const values got from https://github.com/labstack/echo
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

// Gear global values
var (
	GearParamsKey = &contextKey{"Gear-Params-Key"}
	GearLogsKey   = &contextKey{"Gear-Logs-Key"}
)

var nilByte []byte

// Context represents the context of the current HTTP request. It holds request and
// response objects, path, path parameters, data, registered handler and content.Context.
type Context struct {
	ctx        context.Context
	cancelCtx  context.CancelFunc
	app        *Gear
	Req        *http.Request
	Res        *Response
	Host       string
	Method     string
	Path       string
	ended      bool
	query      url.Values
	vals       map[interface{}]interface{}
	afterHooks []Hook
	endHooks   []Hook
}

// NewContext creates an instance of Context. It is useful for testing a middleware.
func NewContext(g *Gear) *Context {
	ctx := &Context{app: g, Res: &Response{}}
	ctx.Res.ctx = ctx
	return ctx
}

// Reset initializes the ctx with http.ResponseWriter and http.Request.
func (ctx *Context) Reset(w http.ResponseWriter, req *http.Request) {
	ctx.Req = req
	ctx.Res.reset(w)
	ctx.ended = false
	if w == nil {
		ctx.ctx = nil
		ctx.vals = nil
		ctx.query = nil
		ctx.afterHooks = nil
		ctx.endHooks = nil
		ctx.cancelCtx = nil
	} else {
		ctx.Host = req.Host
		ctx.Method = req.Method
		ctx.Path = normalizePath(req.URL.Path) // fix "/abc//ef" to "/abc/ef"
		ctx.vals = make(map[interface{}]interface{})
		ctx.ctx, ctx.cancelCtx = context.WithCancel(req.Context())
	}
}

// ----- implement context.Context interface ----- //

// Deadline returns the time when work done on behalf of this context
// should be canceled.
func (ctx *Context) Deadline() (time.Time, bool) {
	return ctx.ctx.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this
// context should be canceled.
func (ctx *Context) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

// Err returns a non-nil error value after Done is closed.
func (ctx *Context) Err() error {
	return ctx.ctx.Err()
}

// Value returns the value associated with this context for key, or nil
// if no value is associated with key. Successive calls to Value with
// the same key returns the same result.
func (ctx *Context) Value(key interface{}) (val interface{}) {
	var ok bool
	if val, ok = ctx.vals[key]; !ok {
		val = ctx.ctx.Value(key)
	}
	return
}

// Cancel cancel the ctx and all it' children context
func (ctx *Context) Cancel() {
	ctx.cancelCtx()
}

// WithCancel returns a copy of the ctx with a new Done channel.
// The returned context's Done channel is closed when the returned cancel function is called or when the parent context's Done channel is closed, whichever happens first.
func (ctx *Context) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx.ctx)
}

// WithDeadline returns a copy of the ctx with the deadline adjusted to be no later than d.
func (ctx *Context) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(ctx.ctx, deadline)
}

// WithTimeout returns WithDeadline(time.Now().Add(timeout)).
func (ctx *Context) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx.ctx, timeout)
}

// WithValue returns a copy of the ctx in which the value associated with key is val.
func (ctx *Context) WithValue(key, val interface{}) context.Context {
	return context.WithValue(ctx.ctx, key, val)
}

// SetValue save a key and value to the ctx.
func (ctx *Context) SetValue(key, val interface{}) {
	ctx.vals[key] = val
}

// IP returns the client's network address based on `X-Forwarded-For`
// or `X-Real-IP` request header.
func (ctx *Context) IP() string {
	ra := ctx.Req.RemoteAddr
	if ip := ctx.Req.Header.Get(HeaderXForwardedFor); ip != "" {
		ra = ip
	} else if ip := ctx.Req.Header.Get(HeaderXRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}
	return ra
}

// Param returns path parameter by name.
func (ctx *Context) Param(key string) (val string) {
	if params := ctx.Value(GearParamsKey); params != nil {
		val, _ = params.(map[string]string)[key]
	}
	return
}

// Query returns the query param for the provided name.
func (ctx *Context) Query(name string) string {
	if ctx.query == nil {
		ctx.query = ctx.Req.URL.Query()
	}
	return ctx.query.Get(name)
}

// Cookie returns the named cookie provided in the request.
func (ctx *Context) Cookie(name string) (*http.Cookie, error) {
	return ctx.Req.Cookie(name)
}

// SetCookie adds a `Set-Cookie` header in HTTP response.
func (ctx *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(ctx.Res.res, cookie)
}

// Cookies returns the HTTP cookies sent with the request.
func (ctx *Context) Cookies() []*http.Cookie {
	return ctx.Req.Cookies()
}

// Get retrieves data from the request Header.
func (ctx *Context) Get(key string) string {
	return ctx.Req.Header.Get(key)
}

// Set saves data to the response Header.
func (ctx *Context) Set(key, value string) {
	ctx.Res.Set(key, value)
}

// Status set a status code to response
func (ctx *Context) Status(code int) {
	if statusText := http.StatusText(code); statusText == "" {
		code = 500
	}
	ctx.Res.Status = code
}

// Type set a content type to response
func (ctx *Context) Type(str string) {
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
		ctx.Res.Del(HeaderContentType)
	} else {
		ctx.Res.Set(HeaderContentType, str)
	}
}

// String set a string to response.
func (ctx *Context) String(str string) {
	ctx.Res.Body = stringToBytes(str)
}

// Error set a error message with status code to response.
// It will end the ctx. The middlewares after current middleware and "after hooks" will not run.
// "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Error(err HTTPError) {
	ctx.ended = true
	ctx.afterHooks = nil // clear afterHooks when error
	http.Error(ctx.Res, err.Error(), err.Status())
}

// HTML set an Html body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) HTML(code int, str string) error {
	ctx.Type("html")
	ctx.End(code, stringToBytes(str))
	return nil
}

// JSON set a JSON body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSON(code int, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.JSONBlob(code, buf)
}

// JSONBlob set a JSON blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONBlob(code int, buf []byte) error {
	ctx.Type("json")
	ctx.End(code, buf)
	return nil
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONP(code int, callback string, val interface{}) error {
	buf, err := json.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.JSONPBlob(code, callback, buf)
}

// JSONPBlob sends a JSONP blob response with status code. It uses `callback`
// to construct the JSONP payload.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) JSONPBlob(code int, callback string, buf []byte) error {
	ctx.Type("js")
	buf = bytes.Join([][]byte{[]byte(callback + "("), buf, []byte(");")}, []byte{})
	ctx.End(code, buf)
	return nil
}

// XML set an XML body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) XML(code int, val interface{}) error {
	buf, err := xml.Marshal(val)
	if err != nil {
		ctx.Status(500)
		return err
	}
	return ctx.XMLBlob(code, buf)
}

// XMLBlob set a XML blob body with status code to response.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) XMLBlob(code int, buf []byte) error {
	ctx.Type("xml")
	ctx.End(code, buf)
	return nil
}

// Render renders a template with data and sends a text/html response with status
// code. Templates can be registered using `app.Renderer = Renderer`.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" (if no error) and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Render(code int, name string, data interface{}) (err error) {
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

// Stream sends a streaming response with status code and content type.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Stream(code int, contentType string, r io.Reader) (err error) {
	ctx.End(code, nilByte)
	ctx.Type(contentType)
	_, err = io.Copy(ctx.Res, r)
	return
}

// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
// client to save the file.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Attachment(name string, content io.ReadSeeker) error {
	return ctx.contentDisposition("attachment", name, content)
}

// Inline sends a response from `io.ReaderSeeker` as inline, opening
// the file in the browser.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Inline(name string, content io.ReadSeeker) error {
	return ctx.contentDisposition("inline", name, content)
}

func (ctx *Context) contentDisposition(dispositionType, name string, content io.ReadSeeker) error {
	ctx.ended = true
	ctx.Set(HeaderContentDisposition, fmt.Sprintf("%s; filename=%s", dispositionType, name))
	http.ServeContent(ctx.Res, ctx.Req, name, time.Time{}, content)
	return nil
}

// Redirect redirects the request with status code.
// It will end the ctx. The middlewares after current middleware will not run.
// "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) Redirect(code int, url string) error {
	ctx.ended = true
	http.Redirect(ctx.Res, ctx.Req, url, code)
	return nil
}

// End end the ctx with bytes and status code optionally.
// After it's called, the rest of middleware handles will not run.
// But "after hooks" and "end hooks" will run normally.
// Note that this will not stop the current handler.
func (ctx *Context) End(code int, buf []byte) {
	ctx.ended = true
	if code != 0 {
		ctx.Status(code)
	}
	if buf != nil {
		ctx.Res.Body = buf
	}
}

// IsEnded return the ctx' ended status.
func (ctx *Context) IsEnded() bool {
	return ctx.ended || ctx.Res.finished
}

// After add a "after hook" to the ctx that will run after app's Middleware.
func (ctx *Context) After(hook Hook) {
	if !ctx.ended { // should not add afterHooks if ctx.ended
		ctx.afterHooks = append(ctx.afterHooks, hook)
	}
}

// OnEnd add a "end hook" to the ctx that will run before response.WriteHeader.
func (ctx *Context) OnEnd(hook Hook) {
	if !ctx.Res.finished { // should not add endHooks if ctx.Res.finished
		ctx.endHooks = append(ctx.endHooks, hook)
	}
}

func (ctx *Context) runAfterHooks() {
	ctx.ended = true // ensure ctx.ended to true
	for _, hook := range ctx.afterHooks {
		if ctx.Res.finished {
			break
		}
		hook(ctx)
	}
	ctx.afterHooks = nil
}

func (ctx *Context) runEndHooks() {
	ctx.Res.finished = true // ensure ctx.Res.finished to true
	for _, hook := range ctx.endHooks {
		if ctx.Res.finished {
			break
		}
		hook(ctx)
	}
	ctx.endHooks = nil
}
