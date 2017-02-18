package main

import (
	"flag"

	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"github.com/teambition/gear/middleware/static"
)

var (
	address = flag.String("addr", "127.0.0.1:3000", `address to listen on.`)
	path    = flag.String("path", "./", `static files path to serve.`)
)

func main() {
	flag.Parse()
	app := gear.New()

	app.UseHandler(logging.Default())
	app.Use(static.New(static.Options{Root: *path}))

	logging.Println("staticgo v1.0.0, created by https://github.com/teambition/gear")
	logging.Printf("listen: %s, serve: %s\n", *address, *path)
	app.Error(app.Listen(*address))
}
