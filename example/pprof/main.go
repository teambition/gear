package main

import (
	"net/http/pprof"

	"github.com/teambition/gear"
)

// go run example/pprof/main.go
func main() {
	app := gear.New()

	router := gear.NewRouter()
	// try: http://127.0.0.1:3000/debug/pprof
	router.Get("/debug/pprof", gear.WrapHandlerFunc(pprof.Index))
	router.Get("/debug/pprof/cmdline", gear.WrapHandlerFunc(pprof.Cmdline))
	router.Get("/debug/pprof/profile", gear.WrapHandlerFunc(pprof.Profile))
	router.Get("/debug/pprof/symbol", gear.WrapHandlerFunc(pprof.Symbol))
	router.Get("/debug/pprof/trace", gear.WrapHandlerFunc(pprof.Trace))

	app.UseHandler(router)
	app.Error(app.Listen(":3000"))
}
