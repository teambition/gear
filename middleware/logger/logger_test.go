package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

// ----- Test Helpers -----
type GearResponse struct {
	*http.Response
}

var DefaultClient = &http.Client{}

func RequestBy(method, url string) (*GearResponse, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}
func NewRequst(method, url string) (*http.Request, error) {
	return http.NewRequest(method, url, nil)
}
func DefaultClientDo(req *http.Request) (*GearResponse, error) {
	res, err := DefaultClient.Do(req)
	return &GearResponse{res}, err
}

type testLogger struct {
	W io.Writer
}

func (logger *testLogger) FromCtx(ctx *gear.Context) Log {
	if any, err := ctx.Any(logger); err == nil {
		return any.(Log)
	}
	log := Log{}
	ctx.SetAny(logger, log)

	log["IP"] = ctx.IP()
	log["Method"] = ctx.Method
	log["URL"] = ctx.Req.URL.String()
	log["Start"] = time.Now()
	log["UserAgent"] = ctx.Get(gear.HeaderUserAgent)
	return log
}

func (logger *testLogger) WriteLog(log Log) {
	// Format: ":Date INFO :JSONString"
	end := time.Now()
	info := map[string]interface{}{
		"IP":        log["IP"],
		"Method":    log["Method"],
		"URL":       log["URL"],
		"UserAgent": log["UserAgent"],
		"Status":    log["Status"],
		"Length":    log["Length"],
		"Data":      log["Data"],
		"Time":      end.Sub(log["Start"].(time.Time)) / 1e6,
	}

	var str string
	switch res, err := json.Marshal(info); err == nil {
	case true:
		str = fmt.Sprintf("%s INFO %s", end.Format(time.RFC3339), bytes.NewBuffer(res).String())
	default:
		str = fmt.Sprintf("%s ERROR %s", end.Format(time.RFC3339), err.Error())
	}
	// Don't block current process.
	go fmt.Fprintln(logger.W, str)
}

func TestGearLogger(t *testing.T) {
	t.Run("Simple log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := &testLogger{&buf}
		app.Use(NewLogger(logger))
		app.Use(func(ctx *gear.Context) error {
			log := logger.FromCtx(ctx)
			log["Data"] = []int{1, 2, 3}
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		log := buf.String()
		assert.Contains(log, time.Now().Format(time.RFC3339)[0:19])
		assert.Contains(log, " INFO ")
		assert.Contains(log, `"Data":[1,2,3]`)
		assert.Contains(log, `"Method":"GET"`)
		assert.Contains(log, `"Status":200`)
		assert.Contains(log, `"UserAgent":`)
		res.Body.Close()
	})

	t.Run("Default log", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("use native color func for windows platform")
		}
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := &DefaultLogger{&buf}
		app.Use(NewLogger(logger))
		app.Use(func(ctx *gear.Context) error {
			log := logger.FromCtx(ctx)
			log["Data"] = []int{1, 2, 3}
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		log := buf.String()

		assert.Contains(log, "\x1b[34;1mGET\x1b[39;22m")
		assert.Contains(log, "\x1b[32;1m200\x1b[39;22m")
		res.Body.Close()
	})

	t.Run("Work with panic", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		var errbuf bytes.Buffer

		app := gear.New()
		app.Set("AppLogger", log.New(&errbuf, "TEST: ", 0))

		logger := &testLogger{&buf}
		app.Use(NewLogger(logger))
		app.Use(func(ctx *gear.Context) (err error) {
			log := logger.FromCtx(ctx)
			log["Data"] = map[string]interface{}{"a": 0}
			panic("Some error")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("POST", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(500, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		log := buf.String()
		assert.Contains(log, time.Now().Format(time.RFC3339)[0:19])
		assert.Contains(log, " INFO ")
		assert.Contains(log, `"Data":{"a":0}`)
		assert.Contains(log, `"Method":"POST"`)
		assert.Contains(log, `"Status":500`)
		assert.Contains(log, `"UserAgent":`)
		assert.Contains(errbuf.String(), "Some error")
		res.Body.Close()
	})

	t.Run("Color", func(t *testing.T) {

		assert := assert.New(t)

		assert.Equal(ColorCodeGreen, ColorStatus(200))
		assert.Equal(ColorCodeGreen, ColorStatus(204))
		assert.Equal(ColorCodeWhite, ColorStatus(304))
		assert.Equal(ColorCodeYellow, ColorStatus(404))
		assert.Equal(ColorCodeRed, ColorStatus(504))

		assert.Equal(ColorCodeBlue, ColorMethod("GET"))
		assert.Equal(ColorCodeMagenta, ColorMethod("HEAD"))
		assert.Equal(ColorCodeCyan, ColorMethod("POST"))
		assert.Equal(ColorCodeYellow, ColorMethod("PUT"))
		assert.Equal(ColorCodeRed, ColorMethod("DELETE"))
		assert.Equal(ColorCodeWhite, ColorMethod("OPTIONS"))
		assert.Equal(ColorCodeWhite, ColorMethod("PATCH"))
	})
}
