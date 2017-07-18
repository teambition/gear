package logging

import (
	"bytes"
	"log"
	"math"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

// ----- Test Helpers -----
func EqualPtr(t *testing.T, a, b interface{}) {
	assert.Equal(t, reflect.ValueOf(a).Pointer(), reflect.ValueOf(b).Pointer())
}

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

func TestGearLog(t *testing.T) {
	t.Run("Log.Format", func(t *testing.T) {
		assert := assert.New(t)

		log := Log{"value": 1}
		str, err := log.Format()
		assert.Nil(err)
		assert.Equal(`{"value":1}`, str)

		log = Log{"value": math.NaN}
		str, err = log.Format()
		assert.NotNil(err)
		assert.Equal("", str)
	})

	t.Run("Log.String", func(t *testing.T) {
		assert := assert.New(t)

		log := Log{"value": 1}
		assert.Equal(`Log{value:1}`, log.String())

		log = Log{"key": "test", "value": 1}
		assert.True(strings.Contains(log.String(), `key:"test"`))
		assert.True(strings.Contains(log.String(), `value:1`))
		assert.True(strings.Contains(log.String(), `, `))
	})

	t.Run("Log.From", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := Log{"key2": true}

		EqualPtr(t, log1, log1.From(log2))
		assert.Equal(Log{"key1": 1, "key2": true}, log1)
	})

	t.Run("Log.Into", func(t *testing.T) {
		assert := assert.New(t)

		log1 := Log{"key1": 1}
		log2 := Log{"key2": true}

		EqualPtr(t, log2, log1.Into(log2))
		assert.Equal(Log{"key2": true, "key1": 1}, log2)
	})
}

func TestGearLogger(t *testing.T) {
	exit = func() {} // overwrite exit function

	t.Run("Default logger", func(t *testing.T) {
		assert := assert.New(t)

		logger := Default()
		assert.Equal(logger.l, DebugLevel)
		assert.Equal(logger.tf, "2006-01-02T15:04:05.999Z")
		assert.Equal(logger.lf, "[%s] %s %s")

		var buf bytes.Buffer

		logger.Out = &buf
		logger.Emerg("Hello")
		assert.True(strings.Index(buf.String(), "Z] EMERG Error") > 0)
		buf.Reset()

		Emerg("Hello1")
		assert.True(strings.Index(buf.String(), "Z] EMERG Error") > 0)
		buf.Reset()

		logger.Alert("Hello")
		assert.True(strings.Index(buf.String(), "Z] ALERT Error") > 0)
		buf.Reset()

		Alert("Hello1")
		assert.True(strings.Index(buf.String(), "Z] ALERT Error") > 0)
		buf.Reset()

		logger.Crit("Hello")
		assert.True(strings.Index(buf.String(), "Z] CRIT Error") > 0)
		buf.Reset()

		Crit("Hello1")
		assert.True(strings.Index(buf.String(), "Z] CRIT Error") > 0)
		buf.Reset()

		logger.Err("Hello")
		assert.True(strings.Index(buf.String(), "Z] ERR Error") > 0)
		buf.Reset()

		Err("Hello1")
		assert.True(strings.Index(buf.String(), "Z] ERR Error") > 0)
		buf.Reset()

		logger.Warning("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] WARNING Hello\n"))
		buf.Reset()

		Warning("Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] WARNING Hello1\n"))
		buf.Reset()

		logger.Warning(Log{"error": "some \n err\r\nor"})
		assert.True(strings.HasSuffix(buf.String(), "Z] WARNING {\"error\":\"some \\n err\\r\\nor\"}\n"))
		buf.Reset()

		logger.Notice("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] NOTICE Hello\n"))
		buf.Reset()

		Notice("Hello\r1\n")
		assert.True(strings.HasSuffix(buf.String(), "Z] NOTICE Hello\\r1\n"))
		buf.Reset()

		logger.Notice(Log{"msg": "some\r\nmsg\n"})
		assert.True(strings.HasSuffix(buf.String(), "Z] NOTICE {\"msg\":\"some\\r\\nmsg\\n\"}\n"))
		buf.Reset()

		logger.Info("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] INFO Hello\n"))
		buf.Reset()

		logger.Info(Log{"name": "gear"})
		assert.True(strings.HasSuffix(buf.String(), "Z] INFO {\"name\":\"gear\"}\n"))
		buf.Reset()

		Info("Hello\r\n1\r\n")
		assert.True(strings.HasSuffix(buf.String(), "Z] INFO Hello\\r\\n1\\r\n"))
		buf.Reset()

		logger.Debug("Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] DEBUG Hello\n"))
		buf.Reset()

		Debug("Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] DEBUG Hello1\n"))
		buf.Reset()

		logger.Debugf(":%s\n", "Hello")
		assert.True(strings.HasSuffix(buf.String(), "Z] DEBUG :Hello\n"))
		buf.Reset()

		Debugf(":%s\n", "Hello1")
		assert.True(strings.HasSuffix(buf.String(), "Z] DEBUG :Hello1\n"))
		buf.Reset()

		assert.Panics(func() {
			logger.Panic("Hello")
		})
		assert.True(strings.Index(buf.String(), "EMERG Error") > 0)
		buf.Reset()

		assert.Panics(func() {
			Panic("Hello1")
		})
		assert.True(strings.Index(buf.String(), "EMERG Error") > 0)
		buf.Reset()

		logger.Fatal("Hello")
		assert.True(strings.Index(buf.String(), "EMERG Error") > 0)
		buf.Reset()

		Fatal("Hello1")
		assert.True(strings.Index(buf.String(), "EMERG Error") > 0)
		buf.Reset()

		logger.Print("Hello")
		assert.Equal(buf.String(), "Hello")
		buf.Reset()

		Print("Hello1")
		assert.Equal(buf.String(), "Hello1")
		buf.Reset()

		logger.Printf(":%s", "Hello")
		assert.Equal(buf.String(), ":Hello")
		buf.Reset()

		Printf(":%s", "Hello1")
		assert.Equal(buf.String(), ":Hello1")
		buf.Reset()

		logger.Println("Hello")
		assert.Equal(buf.String(), "Hello\n")
		buf.Reset()

		Println("Hello1")
		assert.Equal(buf.String(), "Hello1\n")
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "Hello")
		assert.True(strings.HasSuffix(buf.String(), "INFO Hello\n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "")
		assert.True(strings.HasSuffix(buf.String(), "INFO \n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "\n")
		assert.True(strings.HasSuffix(buf.String(), "INFO \n"))
		buf.Reset()

		logger.Output(time.Now(), InfoLevel, "\r")
		assert.True(strings.HasSuffix(buf.String(), "INFO \\r\n"))
		buf.Reset()

	})

	t.Run("logger setting", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		logger := New(&buf)
		assert.Panics(func() {
			var level Level = 8
			logger.SetLevel(level)
		})
		logger.SetLevel(NoticeLevel)
		logger.Info("Hello")
		assert.Equal(buf.String(), "")
		buf.Reset()

		logger.SetLogFormat("%s") // with invalid format
		logger.SetLevel(DebugLevel)
		logger.Info("Hello")
		assert.Equal(strings.Contains(buf.String(), "INFO"), true)
		buf.Reset()
	})
}

func TestGearLoggerMiddleware(t *testing.T) {
	t.Run("Default log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := Default()
		logger.Out = &buf
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			log := logger.FromCtx(ctx)
			if ctx.Path == "/reset" {
				log.Reset()
			} else {
				log["Data"] = []int{1, 2, 3}
			}
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:16])
		assert.Contains(log, "] INFO ")
		assert.Contains(log, `"Data":[1,2,3],`)
		assert.Contains(log, `"Method":"GET",`)
		assert.Contains(log, `"Length":2,`)
		assert.Contains(log, `"Status":200,`)
		res.Body.Close()

		buf.Reset()
		res, err = RequestBy("GET", "http://"+srv.Addr().String()+"/reset")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		assert.Equal(buf.String(), "")
		logger.mu.Unlock()
		res.Body.Close()
	})

	t.Run("Default log with development mode", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("use native color func for windows platform")
		}
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()
		logger := Default(true)
		logger.Out = &buf
		app.UseHandler(logger)
		app.Use(func(ctx *gear.Context) error {
			log := FromCtx(ctx)
			EqualPtr(t, log, logger.FromCtx(ctx))
			return ctx.HTML(200, "OK")
		})
		srv := app.Start()
		defer srv.Close()

		res, err := RequestBy("GET", "http://"+srv.Addr().String())
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()

		assert.Contains(log, "\x1b[32;1m127.0.0.1\x1b[39;22m - -")
		assert.Contains(log, `"GET / `)
		assert.Contains(log, "\x1b[32;1m200\x1b[39;22m")
		res.Body.Close()
	})

	t.Run("custom log", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		app := gear.New()

		logger := New(&buf)
		logger.SetLogInit(func(log Log, ctx *gear.Context) {
			log["IP"] = ctx.IP()
			log["Method"] = ctx.Method
			log["URL"] = ctx.Req.URL.String()
			log["Start"] = time.Now()
			log["UserAgent"] = ctx.Get(gear.HeaderUserAgent)
		})
		logger.SetLogConsume(func(log Log, _ *gear.Context) {
			end := time.Now()
			log["Time"] = end.Sub(log["Start"].(time.Time)) / 1e6
			delete(log, "Start")
			if res, err := log.Format(); err == nil {
				logger.Output(end, InfoLevel, res)
			} else {
				logger.Output(end, WarningLevel, log.String())
			}
		})

		app.UseHandler(logger)
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
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:18])
		assert.Contains(log, "] INFO ")
		assert.Contains(log, `"Data":[1,2,3],`)
		assert.Contains(log, `"Method":"GET",`)
		assert.Contains(log, `"Length":2,`)
		assert.Contains(log, `"Status":200,`)
		assert.Contains(log, `"UserAgent":`)
		assert.Equal(rune(log[len(log)-1]), '\n')
		res.Body.Close()
	})

	t.Run("Work with panic", func(t *testing.T) {
		assert := assert.New(t)

		var buf bytes.Buffer
		var errbuf bytes.Buffer

		app := gear.New()
		app.Set(gear.SetLogger, log.New(&errbuf, "TEST: ", 0))

		logger := New(&buf)
		logger.SetLogInit(func(log Log, ctx *gear.Context) {
			log["IP"] = ctx.IP()
			log["Method"] = ctx.Method
			log["URL"] = ctx.Req.URL.String()
			log["Start"] = time.Now()
			log["UserAgent"] = ctx.Get(gear.HeaderUserAgent)
		})
		logger.SetLogConsume(func(log Log, _ *gear.Context) {
			end := time.Now()
			log["Time"] = end.Sub(log["Start"].(time.Time)) / 1e6
			delete(log, "Start")
			if res, err := log.Format(); err == nil {
				logger.Output(end, InfoLevel, res)
			} else {
				logger.Output(end, WarningLevel, log.String())
			}
		})

		app.UseHandler(logger)
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
		assert.Equal("application/json; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		time.Sleep(10 * time.Millisecond)
		logger.mu.Lock()
		log := buf.String()
		logger.mu.Unlock()
		assert.Contains(log, time.Now().UTC().Format(time.RFC3339)[0:18])
		assert.Contains(log, "] INFO ")
		assert.Contains(log, `"Data":{"a":0}`)
		assert.Contains(log, `"Method":"POST"`)
		assert.Contains(log, `"Status":500`)
		assert.Contains(log, `"UserAgent":`)
		assert.Contains(errbuf.String(), "Some error")
		res.Body.Close()
	})

	t.Run("Color", func(t *testing.T) {
		assert := assert.New(t)

		assert.Equal(ColorGreen, colorStatus(200))
		assert.Equal(ColorGreen, colorStatus(204))
		assert.Equal(ColorCyan, colorStatus(304))
		assert.Equal(ColorYellow, colorStatus(404))
		assert.Equal(ColorRed, colorStatus(504))
	})
}
