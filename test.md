# gear
--
    import "github.com/teambition/gear"


## Usage

```go
const (
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
```
All const values got from https://github.com/labstack/echo MIME types

```go
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
```
Headers

```go
var (
	GearParamsKey = &contextKey{"Gear-Params-Key"}
	GearLogsKey   = &contextKey{"Gear-Logs-Key"}
)
```
Gear global values

#### type Context

```go
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
```

Context represents the context of the current HTTP request. It holds request and
response objects, path, path parameters, data, registered handler and
content.Context.

#### type Gear

```go
type Gear struct {

	// ErrorLog specifies an optional logger for app's errors.
	ErrorLog *log.Logger

	// OnCtxError is error handle for Middleware error.
	OnCtxError func(Context, error) *HTTPError
	Renderer   Renderer
	Server     *http.Server
}
```

Gear is the top-level framework app instance.

#### func  New

```go
func New() *Gear
```
New creates an instance of Gear.

#### func (*Gear) Listen

```go
func (g *Gear) Listen(addr string) error
```
Listen starts the HTTP server.

#### func (*Gear) ListenTLS

```go
func (g *Gear) ListenTLS(addr, certFile, keyFile string) error
```
ListenTLS starts the HTTPS server.

#### func (*Gear) OnError

```go
func (g *Gear) OnError(err error)
```
OnError is default app error handler.

#### func (*Gear) Use

```go
func (g *Gear) Use(handle Middleware)
```
Use uses the given middleware `handle`.

#### func (*Gear) UseHandler

```go
func (g *Gear) UseHandler(h Handler)
```
UseHandler uses a instance that implemented Handler interface.

#### type HTTPError

```go
type HTTPError struct {
	Code int
}
```

HTTPError represents an error that occurred while handling a request.

#### func  NewHTTPError

```go
func NewHTTPError(code int, err string) *HTTPError
```
NewHTTPError creates an instance of HTTPError with status code and error
message.

#### type Handler

```go
type Handler interface {
	Middleware(Context) error
}
```

Handler is the interface that wraps the Middleware function.

#### type Hook

```go
type Hook func(Context)
```

Hook defines a function to process hook.

#### type Middleware

```go
type Middleware func(Context) error
```

Middleware defines a function to process middleware.

#### type Renderer

```go
type Renderer interface {
	Render(Context, io.Writer, string, interface{}) error
}
```

Renderer is the interface that wraps the Render function.

#### type Response

```go
type Response struct {
	Status int    // response Status
	Type   string // response Content-Type
	Body   []byte // response Content
}
```

Response wraps an http.ResponseWriter and implements its interface to be used by
an HTTP handler to construct an HTTP response.

#### func (*Response) Add

```go
func (r *Response) Add(key, value string)
```
Add adds the key, value pair to the header. It appends to any existing values
associated with key.

#### func (*Response) Del

```go
func (r *Response) Del(key string)
```
Del deletes the values associated with key.

#### func (*Response) Get

```go
func (r *Response) Get(key string) string
```
Get gets the first value associated with the given key. If there are no values
associated with the key, Get returns "". To access multiple values of a key,
access the map directly with CanonicalHeaderKey.

#### func (*Response) Header

```go
func (r *Response) Header() http.Header
```
Header returns the header map that will be sent by WriteHeader.

#### func (*Response) Set

```go
func (r *Response) Set(key, value string)
```
Set sets the header entries associated with key to the single element value. It
replaces any existing values associated with key.

#### func (*Response) Write

```go
func (r *Response) Write(buf []byte) (int, error)
```
Write writes the data to the connection as part of an HTTP reply.

#### func (*Response) WriteHeader

```go
func (r *Response) WriteHeader(code int)
```
WriteHeader sends an HTTP response header with status code. If WriteHeader is
not called explicitly, the first call to Write will trigger an implicit
WriteHeader(http.StatusOK). Thus explicit calls to WriteHeader are mainly used
to send error codes.

#### type Router

```go
type Router struct {
	// If enabled, the router automatically replies to OPTIONS requests.
	// Default to true
	HandleOPTIONS bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Default to true
	IsEndpoint bool
}
```

Router is a tire base HTTP request handler for Gear which can be used to
dispatch requests to different handler functions

#### func  NewRouter

```go
func NewRouter(root string, ignoreCase bool) *Router
```
NewRouter returns a new Router instance with root path and ignoreCase option.

#### func (*Router) Del

```go
func (r *Router) Del(pattern string, handle Middleware)
```
Del registers a new DELETE route for a path with matching handler in the router.

#### func (*Router) Delete

```go
func (r *Router) Delete(pattern string, handle Middleware)
```
Delete registers a new DELETE route for a path with matching handler in the
router.

#### func (*Router) Get

```go
func (r *Router) Get(pattern string, handle Middleware)
```
Get registers a new GET route for a path with matching handler in the router.

#### func (*Router) Handle

```go
func (r *Router) Handle(method, pattern string, handle Middleware)
```
Handle registers a new Middleware handler with method and path in the router.

#### func (*Router) Head

```go
func (r *Router) Head(pattern string, handle Middleware)
```
Head registers a new HEAD route for a path with matching handler in the router.

#### func (*Router) Middleware

```go
func (r *Router) Middleware(ctx Context) error
```
Middleware implemented gear.Handler interface

#### func (*Router) Options

```go
func (r *Router) Options(pattern string, handle Middleware)
```
Options registers a new OPTIONS route for a path with matching handler in the
router.

#### func (*Router) Otherwise

```go
func (r *Router) Otherwise(handle Middleware)
```
Otherwise registers a new Middleware handler in the router that will run if
there is no other handler matching.

#### func (*Router) Patch

```go
func (r *Router) Patch(pattern string, handle Middleware)
```
Patch registers a new PATCH route for a path with matching handler in the
router.

#### func (*Router) Post

```go
func (r *Router) Post(pattern string, handle Middleware)
```
Post registers a new POST route for a path with matching handler in the router.

#### func (*Router) Put

```go
func (r *Router) Put(pattern string, handle Middleware)
```
Put registers a new PUT route for a path with matching handler in the router.

#### func (*Router) Use

```go
func (r *Router) Use(handle Middleware)
```
Use registers a new Middleware handler in the router.
