package main

import (
	"net/http/pprof"
	"strings"

	"github.com/teambition/gear"
)

// go run example/pprof/main.go
func main() {
	app := gear.New()

	// try: http://127.0.0.1:3000/debug/pprof
	app.Use(func(ctx *gear.Context) error {
		if strings.HasPrefix(ctx.Path, "/debug/pprof") {
			pprof.Index(ctx.Res, ctx.Req)
		}
		return nil
	})
	app.Error(app.Listen(":3000"))
}
