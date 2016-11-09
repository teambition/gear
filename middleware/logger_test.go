package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

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
	go func() {
		if _, err := fmt.Fprintln(logger.W, str); err != nil {
			panic(err)
		}
	}()
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

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
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

		req := NewRequst()
		res, err := req.Get("http://" + srv.Addr().String())
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

		req := NewRequst()
		res, err := req.Post("http://" + srv.Addr().String())
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

		assert.Equal("\x1b[32;1m200\x1b[39;22m", ColorStatus(200))
		assert.Equal("\x1b[32;1m204\x1b[39;22m", ColorStatus(204))
		assert.Equal("\x1b[37;1m304\x1b[39;22m", ColorStatus(304))
		assert.Equal("\x1b[33;1m404\x1b[39;22m", ColorStatus(404))
		assert.Equal("\x1b[31;1m504\x1b[39;22m", ColorStatus(504))

		assert.Equal("\x1b[34;1mGET\x1b[39;22m", ColorMethod("GET"))
		assert.Equal("\x1b[35;1mHEAD\x1b[39;22m", ColorMethod("HEAD"))
		assert.Equal("\x1b[36;1mPOST\x1b[39;22m", ColorMethod("POST"))
		assert.Equal("\x1b[33;1mPUT\x1b[39;22m", ColorMethod("PUT"))
		assert.Equal("\x1b[31;1mDELETE\x1b[39;22m", ColorMethod("DELETE"))
		assert.Equal("\x1b[37;1mOPTIONS\x1b[39;22m", ColorMethod("OPTIONS"))
		assert.Equal("PATCH", ColorMethod("PATCH"))
	})
}
