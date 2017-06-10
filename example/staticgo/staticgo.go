package main

import (
	"flag"

	"github.com/teambition/compressible-go"
	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"github.com/teambition/gear/middleware/cors"
	"github.com/teambition/gear/middleware/static"
)

var (
	address  = flag.String("addr", "127.0.0.1:3000", `address to listen on.`)
	path     = flag.String("path", "./", `static files path to serve.`)
	certFile = flag.String("certFile", "", `certFile path, used to create TLS static server.`)
	keyFile  = flag.String("keyFile", "", `keyFile path, used to create TLS static server.`)
)

func main() {
	flag.Parse()
	app := gear.New()
	app.Set(gear.SetCompress, compressible.WithThreshold(1024))

	app.UseHandler(logging.Default(true))
	app.Use(cors.New())
	app.Use(static.New(static.Options{Root: *path}))

	logging.Println("staticgo v1.1.0, created by https://github.com/teambition/gear")
	logging.Printf("listen: %s, serve: %s\n", *address, *path)

	if *certFile != "" && *keyFile != "" {
		app.Error(app.ListenTLS(*address, *certFile, *keyFile))
	} else {
		app.Error(app.Listen(*address))
	}
}
