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
	MIMEApplicationYAML                  = "application/yaml"
	MIMEApplicationTOML                  = "application/toml" // https://github.com/toml-lang/toml
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf" // https://tools.ietf.org/html/draft-rfernando-protocol-buffers-00
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = "text/html; charset=utf-8"
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = "text/plain; charset=utf-8"
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
	MIMEApplicationSchemaJSON            = "application/schema+json"
	MIMEApplicationSchemaInstanceJSON    = "application/schema-instance+json"
	MIMEApplicationSchemaJSONLD          = "application/ld+json"
	MIMEApplicationSchemaGraphQL         = "application/graphql"
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
	HeaderXRequestID         = "X-Request-Id"        // Requests
	HeaderXCanary            = "X-Canary"            // Requests, Responses
	HeaderXForwardedScheme   = "X-Forwarded-Scheme"  // Requests
	HeaderXForwardedProto    = "X-Forwarded-Proto"   // Requests
	HeaderXForwardedFor      = "X-Forwarded-For"     // Requests
	HeaderXForwardedHost     = "X-Forwarded-Host"    // Requests
	HeaderXForwardedServer   = "X-Forwarded-Server"  // Requests
	HeaderXRealIP            = "X-Real-Ip"           // Requests
	HeaderXRealScheme        = "X-Real-Scheme"       // Requests

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
	HeaderXCSRFToken                      = "X-CSRF-Token"                        // Responses
	HeaderXHTTPMethodOverride             = "X-HTTP-Method-Override"              // Responses
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

// ErrByStatus returns a gear.Error by http status.
func ErrByStatus(status int) *Error {
	switch status {
	case 400:
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 402:
		return ErrPaymentRequired
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 405:
		return ErrMethodNotAllowed
	case 406:
		return ErrNotAcceptable
	case 407:
		return ErrProxyAuthRequired
	case 408:
		return ErrRequestTimeout
	case 409:
		return ErrConflict
	case 410:
		return ErrGone
	case 411:
		return ErrLengthRequired
	case 412:
		return ErrPreconditionFailed
	case 413:
		return ErrRequestEntityTooLarge
	case 414:
		return ErrRequestURITooLong
	case 415:
		return ErrUnsupportedMediaType
	case 416:
		return ErrRequestedRangeNotSatisfiable
	case 417:
		return ErrExpectationFailed
	case 418:
		return ErrTeapot
	case 421:
		return ErrMisdirectedRequest
	case 422:
		return ErrUnprocessableEntity
	case 423:
		return ErrLocked
	case 424:
		return ErrFailedDependency
	case 426:
		return ErrUpgradeRequired
	case 428:
		return ErrPreconditionRequired
	case 429:
		return ErrTooManyRequests
	case 431:
		return ErrRequestHeaderFieldsTooLarge
	case 451:
		return ErrUnavailableForLegalReasons
	case 499:
		return ErrClientClosedRequest
	case 500:
		return ErrInternalServerError
	case 501:
		return ErrNotImplemented
	case 502:
		return ErrBadGateway
	case 503:
		return ErrServiceUnavailable
	case 504:
		return ErrGatewayTimeout
	case 505:
		return ErrHTTPVersionNotSupported
	case 506:
		return ErrVariantAlsoNegotiates
	case 507:
		return ErrInsufficientStorage
	case 508:
		return ErrLoopDetected
	case 510:
		return ErrNotExtended
	case 511:
		return ErrNetworkAuthenticationRequired
	default:
		return Err.WithCode(status)
	}
}
