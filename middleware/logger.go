package middleware

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/teambition/gear"
)

// Log recodes key-value pairs for logs.
// It will be initialized by NewLogger middleware.
type Log map[string]interface{}

// Logger is a interface for logging. See DefaultLogger.
type Logger interface {
	InitLog(Log, *gear.Context)
	WriteLog(Log)
}

// DefaultLogger is Gear's default logger, useful for development.
// A custom logger example:
//
//  type myLogger struct {
//  	W io.Writer
//  }
//
//  func (l *myLogger) InitLog(log middleware.Log, ctx *gear.Context) {
//  	log["IP"] = ctx.IP()
//  	log["Method"] = ctx.Method
//  	log["URL"] = ctx.Req.URL.String()
//  	log["Start"] = time.Now()
//  	log["UserAgent"] = ctx.Get(gear.HeaderUserAgent)
//  }
//
//  func (l *myLogger) WriteLog(log middleware.Log) {
//  	// Format: ":Date INFO :JSONString"
//  	end := time.Now()
//  	info := map[string]interface{}{
//  		"IP":        log["IP"],
//  		"Method":    log["Method"],
//  		"URL":       log["URL"],
//  		"UserAgent": log["UserAgent"],
//  		"Status":    log["Status"],
//  		"Length":    log["Length"],
//  		"Data":      log["Data"],
//  		"Time":      end.Sub(log["Start"].(time.Time)) / 1e6,
//  	}
//
//  	var str string
//  	switch res, err := json.Marshal(info); err == nil {
//  	case true:
//  		str = fmt.Sprintf("%s INFO %s", end.Format(time.RFC3339), bytes.NewBuffer(res).String())
//  	default:
//  		str = fmt.Sprintf("%s ERROR %s", end.Format(time.RFC3339), err.Error())
//  	}
//  	// Don't block current process.
//  	go func() {
//  		if _, err := fmt.Fprintln(l.W, str); err != nil {
//  			panic(err)
//  		}
//  	}()
//  }
//
type DefaultLogger struct {
	Writer io.Writer
}

// InitLog implements Logger interface
func (d *DefaultLogger) InitLog(log Log, ctx *gear.Context) {
	log["IP"] = ctx.IP()
	log["Method"] = ctx.Method
	log["URL"] = ctx.Req.URL.String()
	log["Start"] = time.Now()
}

// WriteLog implements Logger interface
func (d *DefaultLogger) WriteLog(log Log) {
	// Tiny format: "Method Url Status Content-Length - Response-time ms"
	str := fmt.Sprintf("%s %s %s %d - %.3f ms",
		ColorMethod(log["Method"].(string)),
		log["URL"],
		ColorStatus(log["Status"].(int)),
		log["Length"],
		float64(time.Now().Sub(log["Start"].(time.Time)))/1e6,
	)
	// Don't block current process.
	go func() {
		if _, err := fmt.Fprintln(d.Writer, str); err != nil {
			panic(err)
		}
	}()
}

// NewLogger creates a logger middleware with os.Stdout.
//
//  app := gear.New()
//  logger := &myLogger{os.Stdout}
//  app.Use(middleware.NewLogger(logger))
//  app.Use(func(ctx *gear.Context) error {
//  	any, err := ctx.Any(logger)
//  	if err != nil {
//  		return err
//  	}
//  	log := any.(middleware.Log) // Retrieve the log
//  	log["Data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
//
// `appLogger` Output:
//
//  2016-10-25T08:52:19+08:00 INFO {"Data":{},"IP":"127.0.0.1","Length":2,"Method":"GET","Status":200,"Time":0,"URL":"/","UserAgent":"go-request/0.6.0"}
//
func NewLogger(logger Logger) gear.Middleware {
	return func(ctx *gear.Context) error {
		log := Log{}
		ctx.SetAny(logger, log)
		logger.InitLog(log, ctx)

		// Add a "end hook" to flush logs.
		ctx.OnEnd(func(ctx *gear.Context) {
			log["Status"] = ctx.Res.Status
			log["Length"] = len(ctx.Res.Body)
			logger.WriteLog(log)
		})
		return nil
	}
}

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
