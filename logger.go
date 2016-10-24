package gear

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Color Code https://en.wikipedia.org/wiki/ANSI_escape_code
// 30–37: set text color to one of the colors 0 to 7,
// 40–47: set background color to one of the colors 0 to 7,
// 39: reset text color to default,
// 49: reset background color to default,
// 1: make text bold / bright (this is the standard way to access the bright color variants),
// 22: turn off bold / bright effect, and
// 0: reset all text properties (color, background, brightness, etc.) to their default values.
// For example, one could select bright purple text on a green background (eww!) with the code `\x1B[35;1;42m`
const (
	ColorCodeRed     = 31
	ColorCodeGreen   = 32
	ColorCodeYellow  = 33
	ColorCodeBlue    = 34
	ColorCodeMagenta = 35
	ColorCodeCyan    = 36
	ColorCodeWhite   = 37
	ColorCodeGray    = 90
)

// ColorString convert a string to a color string with color code.
func ColorString(code int, str string) string {
	return fmt.Sprintf("\x1b[%d;1m%s\x1b[39;22m", code, str)
}

// Log represents the key-value pairs for logs.
type Log map[string]interface{}

// Logger is the interface for logging.
type Logger interface {
	Init(*Context)
	Format(Log) string
}

// DefaultLogger is Gear's default logger, useful for development.
type DefaultLogger struct{}

// Init implements Logger interface
func (d *DefaultLogger) Init(ctx *Context) {
	ctx.Log["IP"] = ctx.IP()
	ctx.Log["Method"] = ctx.Method
	ctx.Log["URL"] = ctx.Req.URL.String()
	ctx.Log["startTime"] = time.Now()
}

// Format implements Logger interface
func (d *DefaultLogger) Format(l Log) string {
	// Tiny format: "Method Url Status Content-Length - Response-time ms"
	return fmt.Sprintf("%s %s %s %d - %.3f ms\n",
		ColorMethod(l["Method"].(string)),
		l["URL"],
		ColorStatus(l["Status"].(int)),
		l["Length"],
		float64(time.Now().Sub(l["startTime"].(time.Time)))/1e6,
	)
}

// NewDefaultLogger creates a Gear default logger middleware.
func NewDefaultLogger() Middleware {
	return NewLogger(os.Stdout, &DefaultLogger{})
}

// NewLogger creates a logger middleware with io.Writer and Logger.
func NewLogger(w io.Writer, l Logger) Middleware {
	return func(ctx *Context) error {
		l.Init(ctx)
		ctx.OnEnd(func(ctx *Context) {
			ctx.Log["Status"] = ctx.Res.Status
			ctx.Log["Length"] = 0
			if ctx.Res.Body != nil {
				ctx.Log["Length"] = len(ctx.Res.Body)
			}
			fmt.Fprintf(w, l.Format(ctx.Log))
		})
		return nil
	}
}

// ColorStatus convert a HTTP status code to a color string.
func ColorStatus(code int) string {
	str := fmt.Sprintf("%3d", code)
	switch {
	case code >= 200 && code < 300:
		return ColorString(ColorCodeGreen, str)
	case code >= 300 && code < 400:
		return ColorString(ColorCodeWhite, str)
	case code >= 400 && code < 500:
		return ColorString(ColorCodeYellow, str)
	default:
		return ColorString(ColorCodeRed, str)
	}
}

// ColorMethod convert a HTTP method to a color string.
func ColorMethod(method string) string {
	switch method {
	case http.MethodGet:
		return ColorString(ColorCodeGreen, method)
	case http.MethodHead:
		return ColorString(ColorCodeMagenta, method)
	case http.MethodPost:
		return ColorString(ColorCodeCyan, method)
	case http.MethodPut:
		return ColorString(ColorCodeYellow, method)
	case http.MethodDelete:
		return ColorString(ColorCodeRed, method)
	case http.MethodOptions:
		return ColorString(ColorCodeWhite, method)
	default:
		return method
	}
}
