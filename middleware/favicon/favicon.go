package favicon

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/teambition/gear"
)

// New creates a favicon middleware to serve favicon from the provided directory.
//
//  package main
//
//  import (
//  	"github.com/teambition/gear"
//  	"github.com/teambition/gear/middleware/favicon"
//  )
//
//  func main() {
//  	app := gear.New()
//  	app.Use(favicon.New("./testdata/favicon.ico"))
//  	app.Use(func(ctx *gear.Context) error {
//  		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
//  	})
//  	app.Error(app.Listen(":3000"))
//  }
//
func New(iconpath string) gear.Middleware {
	iconpath = filepath.FromSlash(iconpath)
	if iconpath != "" && iconpath[0] != os.PathSeparator {
		wd, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		iconpath = filepath.Join(wd, iconpath)
	}
	info, _ := os.Stat(iconpath)
	if info == nil || info.IsDir() {
		panic(gear.NewAppError(fmt.Sprintf(`invalid favicon path: "%s"`, iconpath)))
	}
	file, err := ioutil.ReadFile(iconpath)
	if err != nil {
		panic(gear.NewAppError(err.Error()))
	}
	reader := bytes.NewReader(file)

	return func(ctx *gear.Context) (err error) {
		if !strings.HasPrefix(ctx.Path, "/favicon.ico") {
			return
		}
		if ctx.Method != http.MethodGet && ctx.Method != http.MethodHead {
			status := 200
			if ctx.Method != http.MethodOptions {
				status = 405
			}
			ctx.Set(gear.HeaderAllow, "GET, HEAD, OPTIONS")
			return ctx.End(status)
		}
		ctx.Type("image/x-icon")
		http.ServeContent(ctx.Res, ctx.Req, "favicon.ico", info.ModTime(), reader)
		return
	}
}
