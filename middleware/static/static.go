package static

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/teambition/gear"
)

// Options is static middleware options
type Options struct {
	Root        string            // The directory you wish to serve
	Prefix      string            // The url prefix you wish to serve as static request, default to `'/'`.
	StripPrefix bool              // Strip the prefix from URL path, default to `false`.
	Includes    []string          // Optional, a slice of file path to serve, it will ignore Prefix and StripPrefix options.
	Files       map[string][]byte // Optional, a map of File objects to serve.
}

// New creates a static middleware to serves static content from the provided root directory.
//
//  package main
//
//  import (
//  	"github.com/teambition/gear"
//  	"github.com/teambition/gear/middleware/favicon"
//  	"github.com/teambition/gear/middleware/static"
//  )
//
//  func main() {
//  	app := gear.New()
//  	app.Use(favicon.New("./assets/favicon.ico"))
//  	app.Use(static.New(static.Options{
//  		Root:        "./assets",
//  		Prefix:      "/assets",
//  		StripPrefix: false,
//  		Includes:    []string{"/robots.txt"},
//  	}))
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//  	app.Error(app.Listen(":3000"))
//  }
//
func New(opts Options) gear.Middleware {
	modTime := time.Now()
	if opts.Root == "" {
		opts.Root = "."
	}
	root := filepath.FromSlash(opts.Root)
	if root[0] == '.' {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		root = filepath.Join(wd, root)
	}
	info, _ := os.Stat(root)
	if info == nil || !info.IsDir() {
		panic(gear.Err.WithMsgf("invalid root path: %s", root))
	}

	if opts.Prefix == "" {
		opts.Prefix = "/"
	}

	return func(ctx *gear.Context) (err error) {
		path := ctx.Path

		switch {
		case includes(opts.Includes, path): // do nothing
		case strings.HasPrefix(path, opts.Prefix):
			if opts.StripPrefix {
				path = strings.TrimPrefix(path, opts.Prefix)
			}
		default:
			return nil
		}

		if ctx.Method != http.MethodGet && ctx.Method != http.MethodHead {
			status := 200
			if ctx.Method != http.MethodOptions {
				status = 405
			}
			ctx.SetHeader(gear.HeaderAllow, "GET, HEAD, OPTIONS")
			return ctx.End(status)
		}

		if opts.Files != nil {
			if file, ok := opts.Files[path]; ok {
				http.ServeContent(ctx.Res, ctx.Req, path, modTime, bytes.NewReader(file))
				return nil
			}
		}
		path = filepath.Join(root, filepath.FromSlash(path))
		http.ServeFile(ctx.Res, ctx.Req, path)
		return nil
	}
}

func includes(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}
