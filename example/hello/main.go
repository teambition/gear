package main

import (
	"context"
	"fmt"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
)

// go run example/hello/main.go
func main() {
	app := gear.New()

	// Add logging middleware
	app.UseHandler(logging.Default(true))

	// Add router middleware
	router := gear.NewRouter()

	// try: http://127.0.0.1:3000/hello
	router.Get("/hello", func(ctx *gear.Context) error {
		return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
	})

	// try: http://127.0.0.1:3000/test?query=hello
	router.Otherwise(func(ctx *gear.Context) error {
		return ctx.JSON(200, map[string]any{
			"Host":    ctx.Host,
			"Method":  ctx.Method,
			"Path":    ctx.Path,
			"URL":     ctx.Req.URL.String(),
			"Headers": ctx.Req.Header,
		})
	})
	app.UseHandler(router)

	addr := ":3000"
	fmt.Printf("hello server start: %s", addr)
	app.ListenWithContext(gear.ContextWithSignal(context.Background()), addr)
	// starts the HTTPS server.
	// app.ListenWithContext(gear.ContextWithSignal(context.Background()), addr, certFile, keyFile)
}
