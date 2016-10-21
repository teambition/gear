package middleware

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teambition/gear"
)

// StaticOptions is static middleware options
type StaticOptions struct {
	Root        string // The directory you wish to serve
	Prefix      string // The url prefix you wish to serve as static request, default to `'/'`.
	StripPrefix bool   // Strip the prefix from URL path, default to `false`.
}

// NewStatic returns a Static middleware to serves static content from the provided
// root directory.
func NewStatic(opts StaticOptions) gear.Middleware {
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
		panic("Invalid root path: " + root)
	}

	if opts.Prefix == "" {
		opts.Prefix = "/"
	}
	return func(ctx gear.Context) error {
		path := ctx.Path()
		if !strings.HasPrefix(path, opts.Prefix) {
			return nil
		}
		if opts.StripPrefix {
			path = strings.TrimPrefix(path, opts.Prefix)
		}
		path = filepath.Join(root, filepath.FromSlash(path))
		http.ServeFile(ctx.Response(), ctx.Request(), path)
		return nil
	}
}
