package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.Equal(t, "text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		log := buf.String()
		fmt.Println(log)
		require.Contains(t, log, time.Now().Format(time.RFC3339)[0:19])
		require.Contains(t, log, " INFO ")
		require.Contains(t, log, `"Data":[1,2,3]`)
		require.Contains(t, log, `"Method":"GET"`)
		require.Contains(t, log, `"Status":200`)
		require.Contains(t, log, `"UserAgent":`)
		res.Body.Close()
	})

	t.Run("Work with panic", func(t *testing.T) {
		var buf bytes.Buffer
		var errbuf bytes.Buffer

		app := gear.New()
		logger := &testLogger{&buf}
		app.ErrorLog = log.New(&errbuf, "TEST: ", 0)
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
		require.Nil(t, err)
		require.Equal(t, 500, res.StatusCode)
		require.Equal(t, "text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		log := buf.String()
		require.Contains(t, log, time.Now().Format(time.RFC3339)[0:19])
		require.Contains(t, log, " INFO ")
		require.Contains(t, log, `"Data":{"a":0}`)
		require.Contains(t, log, `"Method":"POST"`)
		require.Contains(t, log, `"Status":500`)
		require.Contains(t, log, `"UserAgent":`)
		require.Contains(t, errbuf.String(), "Some error")
		res.Body.Close()
	})
}
