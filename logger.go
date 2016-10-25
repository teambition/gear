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

// Logger is a interface for logging. See DefaultLogger.
type Logger interface {
	Init(*Context)
	Format(Log) string
}

// DefaultLogger is Gear's default logger, useful for development.
//
//  type appLogger struct{}
//
//  func (l *appLogger) Init(ctx *Context) {
//  	ctx.Log["IP"] = ctx.IP()
//  	ctx.Log["Method"] = ctx.Method
//  	ctx.Log["URL"] = ctx.Req.URL.String()
//  	ctx.Log["Start"] = time.Now()
//  	ctx.Log["UserAgent"] = ctx.Get(HeaderUserAgent)
//  }
//
//  func (l *appLogger) Format(log Log) string {
//  	// Format: ":Date INFO :JSONInfo"
//  	end := time.Now()
//  	info := map[string]interface{}{
// 			"IP":        log["IP"],
// 			"Method":    log["Method"],
// 			"URL":       log["URL"],
// 			"UserAgent": log["UserAgent"],
// 			"Status":    log["Status"],
// 			"Length":    log["Length"],
// 			"Data":      log["Data"],
// 			"Time":      end.Sub(log["Start"].(time.Time)) / 1e6,
// 		}
// 		res, err := json.Marshal(info)
// 		if err != nil {
// 			return fmt.Sprintf("%s ERROR %s", end.Format(time.RFC3339), err.Error())
// 		}
// 		return fmt.Sprintf("%s INFO %s", end.Format(time.RFC3339), bytes.NewBuffer(res).String())
// }
//
type DefaultLogger struct{}

// Init implements Logger interface
func (d *DefaultLogger) Init(ctx *Context) {
	ctx.Log["IP"] = ctx.IP()
	ctx.Log["Method"] = ctx.Method
	ctx.Log["URL"] = ctx.Req.URL.String()
	ctx.Log["Start"] = time.Now()
}

// Format implements Logger interface
func (d *DefaultLogger) Format(log Log) string {
	// Tiny format: "Method Url Status Content-Length - Response-time ms"
	return fmt.Sprintf("%s %s %s %d - %.3f ms",
		ColorMethod(log["Method"].(string)),
		log["URL"],
		ColorStatus(log["Status"].(int)),
		log["Length"],
		float64(time.Now().Sub(log["Start"].(time.Time)))/1e6,
	)
}

// NewDefaultLogger creates a Gear default logger middleware.
//
//  app.Use(gear.NewDefaultLogger())
//
func NewDefaultLogger() Middleware {
	return NewLogger(os.Stdout, &DefaultLogger{})
}

// NewLogger creates a logger middleware with io.Writer and Logger.
//
//  app := New()
//  app.Use(NewLogger(os.Stdout, &appLogger{}))
//  app.Use(func(ctx *Context) (err error) {
//  	ctx.Log["Data"] = map[string]interface{}{}
//  	return ctx.HTML(200, "OK")
//  })
//
// `appLogger` Output:
//
//  2016-10-25T08:52:19+08:00 INFO {"Data":{},"IP":"127.0.0.1","Length":2,"Method":"GET","Status":200,"Time":0,"URL":"/","UserAgent":"go-request/0.6.0"}
func NewLogger(w io.Writer, l Logger) Middleware {
	return func(ctx *Context) error {
		ctx.Log = make(Log)

		l.Init(ctx)
		ctx.OnEnd(func(ctx *Context) {
			ctx.Log["Status"] = ctx.Res.Status
			ctx.Log["Length"] = len(ctx.Res.Body)
			if _, err := fmt.Fprintln(w, l.Format(ctx.Log)); err != nil {
				panic(err) // will be recovered by serveHandler
			}
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
		return ColorString(ColorCodeBlue, method)
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
