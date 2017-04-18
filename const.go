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
	GearError = Error{Code: http.StatusInternalServerError, Err: "Gear Error"}

	HTTPErrBadRequest                   = *GearError.WithCode(http.StatusBadRequest)
	HTTPErrUnauthorized                 = *GearError.WithCode(http.StatusUnauthorized)
	HTTPErrPaymentRequired              = *GearError.WithCode(http.StatusPaymentRequired)
	HTTPErrForbidden                    = *GearError.WithCode(http.StatusForbidden)
	HTTPErrNotFound                     = *GearError.WithCode(http.StatusNotFound)
	HTTPErrMethodNotAllowed             = *GearError.WithCode(http.StatusMethodNotAllowed)
	HTTPErrNotAcceptable                = *GearError.WithCode(http.StatusNotAcceptable)
	HTTPErrProxyAuthRequired            = *GearError.WithCode(http.StatusProxyAuthRequired)
	HTTPErrRequestTimeout               = *GearError.WithCode(http.StatusRequestTimeout)
	HTTPErrConflict                     = *GearError.WithCode(http.StatusConflict)
	HTTPErrGone                         = *GearError.WithCode(http.StatusGone)
	HTTPErrLengthRequired               = *GearError.WithCode(http.StatusLengthRequired)
	HTTPErrPreconditionFailed           = *GearError.WithCode(http.StatusPreconditionFailed)
	HTTPErrRequestEntityTooLarge        = *GearError.WithCode(http.StatusRequestEntityTooLarge)
	HTTPErrRequestURITooLong            = *GearError.WithCode(http.StatusRequestURITooLong)
	HTTPErrUnsupportedMediaType         = *GearError.WithCode(http.StatusUnsupportedMediaType)
	HTTPErrRequestedRangeNotSatisfiable = *GearError.WithCode(http.StatusRequestedRangeNotSatisfiable)
	HTTPErrExpectationFailed            = *GearError.WithCode(http.StatusExpectationFailed)
	HTTPErrTeapot                       = *GearError.WithCode(http.StatusTeapot)
	HTTPErrUnprocessableEntity          = *GearError.WithCode(http.StatusUnprocessableEntity)
	HTTPErrLocked                       = *GearError.WithCode(http.StatusLocked)
	HTTPErrFailedDependency             = *GearError.WithCode(http.StatusFailedDependency)
	HTTPErrUpgradeRequired              = *GearError.WithCode(http.StatusUpgradeRequired)
	HTTPErrPreconditionRequired         = *GearError.WithCode(http.StatusPreconditionRequired)
	HTTPErrTooManyRequests              = *GearError.WithCode(http.StatusTooManyRequests)
	HTTPErrRequestHeaderFieldsTooLarge  = *GearError.WithCode(http.StatusRequestHeaderFieldsTooLarge)
	HTTPErrUnavailableForLegalReasons   = *GearError.WithCode(http.StatusUnavailableForLegalReasons)

	HTTPErrInternalServerError           = *GearError.WithCode(http.StatusInternalServerError)
	HTTPErrNotImplemented                = *GearError.WithCode(http.StatusNotImplemented)
	HTTPErrBadGateway                    = *GearError.WithCode(http.StatusBadGateway)
	HTTPErrServiceUnavailable            = *GearError.WithCode(http.StatusServiceUnavailable)
	HTTPErrGatewayTimeout                = *GearError.WithCode(http.StatusGatewayTimeout)
	HTTPErrHTTPVersionNotSupported       = *GearError.WithCode(http.StatusHTTPVersionNotSupported)
	HTTPErrVariantAlsoNegotiates         = *GearError.WithCode(http.StatusVariantAlsoNegotiates)
	HTTPErrInsufficientStorage           = *GearError.WithCode(http.StatusInsufficientStorage)
	HTTPErrLoopDetected                  = *GearError.WithCode(http.StatusLoopDetected)
	HTTPErrNotExtended                   = *GearError.WithCode(http.StatusNotExtended)
	HTTPErrNetworkAuthenticationRequired = *GearError.WithCode(http.StatusNetworkAuthenticationRequired)
)
