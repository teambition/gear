package gear

import "net/http"

// MIME types
const (
	// Got from https://github.com/labstack/echo
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = "application/json; charset=utf-8"
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = "application/javascript; charset=utf-8"
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = "application/xml; charset=utf-8"
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = "text/html; charset=utf-8"
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = "text/plain; charset=utf-8"
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

// HTTP Header Fields
const (
	HeaderAccept             = "Accept"              // Requests, Responses
	HeaderAcceptCharset      = "Accept-Charset"      // Requests
	HeaderAcceptEncoding     = "Accept-Encoding"     // Requests
	HeaderAcceptLanguage     = "Accept-Language"     // Requests
	HeaderAuthorization      = "Authorization"       // Requests
	HeaderCacheControl       = "Cache-Control"       // Requests, Responses
	HeaderContentLength      = "Content-Length"      // Requests, Responses
	HeaderContentMD5         = "Content-MD5"         // Requests, Responses
	HeaderContentType        = "Content-Type"        // Requests, Responses
	HeaderIfMatch            = "If-Match"            // Requests
	HeaderIfModifiedSince    = "If-Modified-Since"   // Requests
	HeaderIfNoneMatch        = "If-None-Match"       // Requests
	HeaderIfRange            = "If-Range"            // Requests
	HeaderIfUnmodifiedSince  = "If-Unmodified-Since" // Requests
	HeaderMaxForwards        = "Max-Forwards"        // Requests
	HeaderProxyAuthorization = "Proxy-Authorization" // Requests
	HeaderPragma             = "Pragma"              // Requests, Responses
	HeaderRange              = "Range"               // Requests
	HeaderReferer            = "Referer"             // Requests
	HeaderUserAgent          = "User-Agent"          // Requests
	HeaderTE                 = "TE"                  // Requests
	HeaderVia                = "Via"                 // Requests
	HeaderWarning            = "Warning"             // Requests, Responses
	HeaderCookie             = "Cookie"              // Requests
	HeaderOrigin             = "Origin"              // Requests
	HeaderAcceptDatetime     = "Accept-Datetime"     // Requests
	HeaderXRequestedWith     = "X-Requested-With"    // Requests

	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"      // Responses
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"     // Responses
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"     // Responses
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials" // Responses
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"    // Responses
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"           // Responses
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"    // Responses
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"   // Responses
	HeaderAcceptPatch                   = "Accept-Patch"                     // Responses
	HeaderAcceptRanges                  = "Accept-Ranges"                    // Responses
	HeaderAllow                         = "Allow"                            // Responses
	HeaderContentEncoding               = "Content-Encoding"                 // Responses
	HeaderContentLanguage               = "Content-Language"                 // Responses
	HeaderContentLocation               = "Content-Location"                 // Responses
	HeaderContentDisposition            = "Content-Disposition"              // Responses
	HeaderContentRange                  = "Content-Range"                    // Responses
	HeaderETag                          = "ETag"                             // Responses
	HeaderExpires                       = "Expires"                          // Responses
	HeaderLastModified                  = "Last-Modified"                    // Responses
	HeaderLink                          = "Link"                             // Responses
	HeaderLocation                      = "Location"                         // Responses
	HeaderP3P                           = "P3P"                              // Responses
	HeaderProxyAuthenticate             = "Proxy-Authenticate"               // Responses
	HeaderRefresh                       = "Refresh"                          // Responses
	HeaderRetryAfter                    = "Retry-After"                      // Responses
	HeaderServer                        = "Server"                           // Responses
	HeaderSetCookie                     = "Set-Cookie"                       // Responses
	HeaderStrictTransportSecurity       = "Strict-Transport-Security"        // Responses
	HeaderTransferEncoding              = "Transfer-Encoding"                // Responses
	HeaderUpgrade                       = "Upgrade"                          // Responses
	HeaderVary                          = "Vary"                             // Responses
	HeaderWWWAuthenticate               = "WWW-Authenticate"                 // Responses
	HeaderPublicKeyPins                 = "Public-Key-Pins"                  // Responses
	HeaderPublicKeyPinsReportOnly       = "Public-Key-Pins-Report-Only"      // Responses
	HeaderRefererPolicy                 = "Referrer-Policy"                  // Responses

	// Common Non-Standard Response Headers
	HeaderXFrameOptions                   = "X-Frame-Options"                     // Responses
	HeaderXXSSProtection                  = "X-XSS-Protection"                    // Responses
	HeaderContentSecurityPolicy           = "Content-Security-Policy"             // Responses
	HeaderContentSecurityPolicyReportOnly = "Content-Security-Policy-Report-Only" // Responses
	HeaderXContentSecurityPolicy          = "X-Content-Security-Policy"           // Responses
	HeaderXWebKitCSP                      = "X-WebKit-CSP"                        // Responses
	HeaderXContentTypeOptions             = "X-Content-Type-Options"              // Responses
	HeaderXPoweredBy                      = "X-Powered-By"                        // Responses
	HeaderXUACompatible                   = "X-UA-Compatible"                     // Responses
	HeaderXForwardedProto                 = "X-Forwarded-Proto"                   // Responses
	HeaderXHTTPMethodOverride             = "X-HTTP-Method-Override"              // Responses
	HeaderXForwardedFor                   = "X-Forwarded-For"                     // Responses
	HeaderXRealIP                         = "X-Real-IP"                           // Responses
	HeaderXCSRFToken                      = "X-CSRF-Token"                        // Responses
	HeaderXDNSPrefetchControl             = "X-DNS-Prefetch-Control"              // Responses
	HeaderXDownloadOptions                = "X-Download-Options"                  // Responses
)

// Predefined errors
var (
	Err = &Error{Code: http.StatusInternalServerError, Err: "Error"}

	// https://golang.org/pkg/net/http/#pkg-constants
	ErrBadRequest                    = Err.WithCode(http.StatusBadRequest).WithErr("BadRequest")
	ErrUnauthorized                  = Err.WithCode(http.StatusUnauthorized).WithErr("Unauthorized")
	ErrPaymentRequired               = Err.WithCode(http.StatusPaymentRequired).WithErr("PaymentRequired")
	ErrForbidden                     = Err.WithCode(http.StatusForbidden).WithErr("Forbidden")
	ErrNotFound                      = Err.WithCode(http.StatusNotFound).WithErr("NotFound")
	ErrMethodNotAllowed              = Err.WithCode(http.StatusMethodNotAllowed).WithErr("MethodNotAllowed")
	ErrNotAcceptable                 = Err.WithCode(http.StatusNotAcceptable).WithErr("NotAcceptable")
	ErrProxyAuthRequired             = Err.WithCode(http.StatusProxyAuthRequired).WithErr("ProxyAuthenticationRequired")
	ErrRequestTimeout                = Err.WithCode(http.StatusRequestTimeout).WithErr("RequestTimeout")
	ErrConflict                      = Err.WithCode(http.StatusConflict).WithErr("Conflict")
	ErrGone                          = Err.WithCode(http.StatusGone).WithErr("Gone")
	ErrLengthRequired                = Err.WithCode(http.StatusLengthRequired).WithErr("LengthRequired")
	ErrPreconditionFailed            = Err.WithCode(http.StatusPreconditionFailed).WithErr("PreconditionFailed")
	ErrRequestEntityTooLarge         = Err.WithCode(http.StatusRequestEntityTooLarge).WithErr("RequestEntityTooLarge")
	ErrRequestURITooLong             = Err.WithCode(http.StatusRequestURITooLong).WithErr("RequestURITooLong")
	ErrUnsupportedMediaType          = Err.WithCode(http.StatusUnsupportedMediaType).WithErr("UnsupportedMediaType")
	ErrRequestedRangeNotSatisfiable  = Err.WithCode(http.StatusRequestedRangeNotSatisfiable).WithErr("RequestedRangeNotSatisfiable")
	ErrExpectationFailed             = Err.WithCode(http.StatusExpectationFailed).WithErr("ExpectationFailed")
	ErrTeapot                        = Err.WithCode(http.StatusTeapot).WithErr("Teapot")
	ErrMisdirectedRequest            = Err.WithCode(421).WithErr("MisdirectedRequest")
	ErrUnprocessableEntity           = Err.WithCode(http.StatusUnprocessableEntity).WithErr("UnprocessableEntity")
	ErrLocked                        = Err.WithCode(http.StatusLocked).WithErr("Locked")
	ErrFailedDependency              = Err.WithCode(http.StatusFailedDependency).WithErr("FailedDependency")
	ErrUpgradeRequired               = Err.WithCode(http.StatusUpgradeRequired).WithErr("UpgradeRequired")
	ErrPreconditionRequired          = Err.WithCode(http.StatusPreconditionRequired).WithErr("PreconditionRequired")
	ErrTooManyRequests               = Err.WithCode(http.StatusTooManyRequests).WithErr("TooManyRequests")
	ErrRequestHeaderFieldsTooLarge   = Err.WithCode(http.StatusRequestHeaderFieldsTooLarge).WithErr("RequestHeaderFieldsTooLarge")
	ErrUnavailableForLegalReasons    = Err.WithCode(http.StatusUnavailableForLegalReasons).WithErr("UnavailableForLegalReasons")
	ErrClientClosedRequest           = Err.WithCode(499).WithErr("ClientClosedRequest")
	ErrInternalServerError           = Err.WithCode(http.StatusInternalServerError).WithErr("InternalServerError")
	ErrNotImplemented                = Err.WithCode(http.StatusNotImplemented).WithErr("NotImplemented")
	ErrBadGateway                    = Err.WithCode(http.StatusBadGateway).WithErr("BadGateway")
	ErrServiceUnavailable            = Err.WithCode(http.StatusServiceUnavailable).WithErr("ServiceUnavailable")
	ErrGatewayTimeout                = Err.WithCode(http.StatusGatewayTimeout).WithErr("GatewayTimeout")
	ErrHTTPVersionNotSupported       = Err.WithCode(http.StatusHTTPVersionNotSupported).WithErr("HTTPVersionNotSupported")
	ErrVariantAlsoNegotiates         = Err.WithCode(http.StatusVariantAlsoNegotiates).WithErr("VariantAlsoNegotiates")
	ErrInsufficientStorage           = Err.WithCode(http.StatusInsufficientStorage).WithErr("InsufficientStorage")
	ErrLoopDetected                  = Err.WithCode(http.StatusLoopDetected).WithErr("LoopDetected")
	ErrNotExtended                   = Err.WithCode(http.StatusNotExtended).WithErr("NotExtended")
	ErrNetworkAuthenticationRequired = Err.WithCode(http.StatusNetworkAuthenticationRequired).WithErr("NetworkAuthenticationRequired")
)
