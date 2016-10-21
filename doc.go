// Package gear is a pithy and powerful web framework for Go, work with context.Context and middleware, like koajs/toajs.

/*
package main

import (
	"github.com/teambition/gear"
	"github.com/teambition/gear/middleware"
)

func main() {
	// Create app
	app := gear.New()
	// Add a static middleware
	// http://localhost:3000/middleware/static.go
	app.Use(middleware.NewStatic(middleware.StaticOptions{
		Root:        "./middleware",
		Prefix:      "/middleware",
		StripPrefix: true,
	}))
	// Add some middleware to app

	app.Use(func(ctx gear.Context) (err error) {
		// fmt.Println(ctx.IP(), ctx.Method(), ctx.Path())
		// Do something...
		return
	})

	// Create views router
	ViewRouter := gear.NewRouter("", true)
	// Matched:
	// "http://localhost:3000"
	ViewRouter.Get("/", func(ctx gear.Context) (err error) {
		ctx.HTML(200, "<h1>Hello, Gear!</h1>")
		return
	})
	// Matched:
	// "http://localhost:3000/view/abc"
	// "http://localhost:3000/view/123"
	ViewRouter.Get("/view/:view", func(ctx gear.Context) (err error) {
		if view := ctx.Param("view"); view == "" {
			ctx.End(400, "Invalid view")
		} else {
			ctx.HTML(200, "View: "+view)
		}
		return
	})
	// Matched:
	// "http://localhost:3000/abc"
	// "http://localhost:3000/abc/efg"
	ViewRouter.Get("/:others*", func(ctx gear.Context) (err error) {
		if others := ctx.Param("others"); others == "" {
			ctx.End(400, "Invalid path")
		} else {
			ctx.HTML(200, "Request path: /"+others)
		}
		return
	})

	// Create API router
	APIRouter := gear.NewRouter("/api", true)
	// Matched:
	// "http://localhost:3000/api/user/abc"
	// "http://localhost:3000/abc/user/123"
	APIRouter.Get("/user/:id", func(ctx gear.Context) (err error) {
		if id := ctx.Param("id"); id == "" {
			ctx.End(400, "Invalid user id")
		} else {
			ctx.JSON(200, map[string]string{
				"Method": ctx.Method(),
				"Path":   ctx.Path(),
				"UserID": id,
			})
		}
		return
	})

	// Must add APIRouter first.
	app.UseHandler(APIRouter)
	app.UseHandler(ViewRouter)
	// Start app at 3000
	app.OnError(app.Listen(":3000"))
}
*/

package gear
