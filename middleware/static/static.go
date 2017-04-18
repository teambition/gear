package static

import (
	"bytes"
	"fmt"
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
//  	app.Use(favicon.New("./testdata/favicon.ico"))
//  	app.Use(static.New(static.Options{
//  		Root:        "./testdata",
//  		Prefix:      "/",
//  		StripPrefix: false,
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
		panic(gear.GearError.WithMsg(fmt.Sprintf("invalid root path: %s", root)))
	}

	if opts.Prefix == "" {
		opts.Prefix = "/"
	}

	return func(ctx *gear.Context) (err error) {
		path := ctx.Path
		if !strings.HasPrefix(path, opts.Prefix) {
			return nil
		}

		if ctx.Method != http.MethodGet && ctx.Method != http.MethodHead {
			status := 200
			if ctx.Method != http.MethodOptions {
				status = 405
			}
			ctx.Set(gear.HeaderAllow, "GET, HEAD, OPTIONS")
			return ctx.End(status)
		}

		if opts.StripPrefix {
			path = strings.TrimPrefix(path, opts.Prefix)
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
