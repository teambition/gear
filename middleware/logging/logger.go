package logging

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"io"

	"github.com/teambition/gear"
)

// Log records key-value pairs for structured logging.
// It will be initialized by New middleware.
type Log map[string]interface{}

// Logger is a interface for logging. See DefaultLogger.
type Logger interface {
	// FromCtx retrieve the log instance from the ctx with ctx.Any.
	// if log instance not exists, FromCtx should create one and save it to the ctx with ctx.SetAny.
	// Here also some initialization work run after created. See DefaultLogger.
	FromCtx(*gear.Context) Log

	// WriteLog will be called on a "end hook". WriteLog should write the log to underlayer logging system.
	WriteLog(Log)
}

// DefaultLogger is Gear's default logger, useful for development.
// A custom logger example:
//
//  type myLogger struct {
//  	Writer io.Writer
//  }
//
//  func (logger *myLogger) FromCtx(ctx *gear.Context) Log {
//  	if any, err := ctx.Any(logger); err == nil {
//  		return any.(Log)
//  	}
//  	log := Log{}
//  	ctx.SetAny(logger, log)
//
//  	log["IP"] = ctx.IP()
//  	log["Method"] = ctx.Method
//  	log["URL"] = ctx.Req.URL.String()
//  	log["Start"] = time.Now()
//  	log["UserAgent"] = ctx.Get(gear.HeaderUserAgent)
//  	return log
//  }
//
//  func (logger *myLogger) WriteLog(log middleware.Log) {
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
//  	fmt.Fprintln(logger.Writer, str)
//  }
//
type DefaultLogger struct {
	Writer io.Writer
}

// FromCtx implements Logger interface
func (logger *DefaultLogger) FromCtx(ctx *gear.Context) Log {
	if any, err := ctx.Any(logger); err == nil {
		return any.(Log)
	}
	log := Log{}
	ctx.SetAny(logger, log)

	log["IP"] = ctx.IP()
	log["Method"] = ctx.Method
	log["URL"] = ctx.Req.URL.String()
	log["Start"] = time.Now()
	return log
}

// WriteLog implements Logger interface
func (logger *DefaultLogger) WriteLog(log Log) {
	method := log["Method"].(string)
	FprintWithColor(logger.Writer, method, ColorMethod(method))
	fmt.Fprint(logger.Writer, " ")

	FprintWithColor(logger.Writer, log["URL"].(string), ColorGray)
	fmt.Fprint(logger.Writer, " ")

	status := log["Status"].(int)
	FprintWithColor(logger.Writer, strconv.Itoa(status), ColorStatus(status))
	fmt.Fprint(logger.Writer, " ")

	length := log["Length"].(int)
	fmt.Fprint(logger.Writer, strconv.Itoa(length)+" ")

	start := fmt.Sprintf(" - %.3f ms", float64(time.Now().Sub(log["Start"].(time.Time)))/1e6)
	fmt.Fprintln(logger.Writer, start)
}

// ColorStatus ...
func ColorStatus(code int) ColorType {
	switch {
	case code >= 200 && code < 300:
		return ColorGreen
	case code >= 300 && code < 400:
		return ColorWhite
	case code >= 400 && code < 500:
		return ColorYellow
	default:
		return ColorRed
	}
}

// ColorMethod ...
func ColorMethod(method string) ColorType {
	switch method {
	case http.MethodGet:
		return ColorBlue
	case http.MethodHead:
		return ColorMagenta
	case http.MethodPost:
		return ColorCyan
	case http.MethodPut:
		return ColorYellow
	case http.MethodDelete:
		return ColorRed
	case http.MethodOptions:
		return ColorWhite
	default:
		return ColorWhite
	}
}

// New creates a logging middleware with a Logger instance.
//
//  app := gear.New()
//  logger := &myLogger{os.Stdout}
//  app.Use(logging.New(logger))
//  app.Use(func(ctx *gear.Context) error {
//  	log := logger.FromCtx(ctx)
//  	log["Data"] = []int{1, 2, 3}
//  	return ctx.HTML(200, "OK")
//  })
// `appLogger` Output:
//
//  2016-10-25T08:52:19+08:00 INFO {"Data":{},"IP":"127.0.0.1","Length":2,"Method":"GET","Status":200,"Time":0,"URL":"/","UserAgent":"go-request/0.6.0"}
func New(logger Logger) gear.Middleware {
	return func(ctx *gear.Context) error {
		// Add a "end hook" to flush logs.
		ctx.OnEnd(func() {
			log := logger.FromCtx(ctx)
			log["Length"] = len(ctx.Res.Body)
			log["Status"] = ctx.Res.Status
			log["Type"] = ctx.Res.Type

			// Don't block current process.
			go logger.WriteLog(log)
		})
		return nil
	}
}
