package static

import (
	"io/ioutil"
	"net/http"
	"testing"

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

func TestGearMiddlewareStatic(t *testing.T) {
	assert.Panics(t, func() {
		New(Options{
			Root:        "../../testdata1",
			Prefix:      "/",
			StripPrefix: false,
		})
	})
	assert.NotPanics(t, func() {
		New(Options{
			Root:        "",
			Prefix:      "",
			StripPrefix: true,
		})
	})

	app := gear.New()
	app.Set(gear.SetCompress, &gear.DefaultCompress{})

	app.Use(New(Options{
		Root:        "../../testdata",
		Prefix:      "/static",
		StripPrefix: true,
	}))
	app.Use(New(Options{
		Root:        "../../testdata",
		Prefix:      "/",
		StripPrefix: false,
	}))
	srv := app.Start()
	defer app.Close()

	t.Run("GET", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("GET with StripPrefix", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/static/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("HEAD", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("HEAD", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("OPTIONS", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("OPTIONS", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other method", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("PATCH", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(405, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("GET, HEAD, OPTIONS", res.Header.Get(gear.HeaderAllow))
		res.Body.Close()
	})

	t.Run("Other file", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/favicon.ico")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("image/x-icon", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("Should compress", func(t *testing.T) {
		assert := assert.New(t)

		req, _ := NewRequst("GET", "http://"+srv.Addr().String()+"/README.md")
		req.Header.Set("Accept-Encoding", "gzip, deflate")
		res, err := DefaultClientDo(req)
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		assert.Equal("gzip", res.Header.Get(gear.HeaderContentEncoding))
		res.Body.Close()
	})

	t.Run("404", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/none.html")
		assert.Nil(err)
		assert.Equal(404, res.StatusCode)
		assert.Equal("text/plain; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})
}

func TestGearMiddlewareStaticWithFileMap(t *testing.T) {
	file, err := ioutil.ReadFile("../../testdata/hello.html")
	if err != nil {
		panic(gear.Err.WithMsg(err.Error()))
	}

	app := gear.New()
	app.Use(New(Options{
		Root: "../../testdata",
		Files: map[string][]byte{
			"/hello_cache.html": file,
		},
	}))
	srv := app.Start()
	defer app.Close()

	t.Run("GET from FileMap", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/hello_cache.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})

	t.Run("GET from system", func(t *testing.T) {
		assert := assert.New(t)

		res, err := RequestBy("GET", "http://"+srv.Addr().String()+"/hello.html")
		assert.Nil(err)
		assert.Equal(200, res.StatusCode)
		assert.Equal("text/html; charset=utf-8", res.Header.Get(gear.HeaderContentType))
		res.Body.Close()
	})
}
