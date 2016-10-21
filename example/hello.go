package main

import (
	"fmt"

	"github.com/teambition/gear"
)

func main() {
	// Create app
	app := gear.New()

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
		if view, ok := ctx.Param("view"); !ok {
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
		if others, ok := ctx.Param("others"); !ok {
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
		if id, ok := ctx.Param("id"); !ok {
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

	// Add some middleware to app
	app.Use(func(ctx gear.Context) (err error) {
		fmt.Println(ctx.IP(), ctx.Method(), ctx.Path())
		// Do something...
		return
	})
	// Must add APIRouter first.
	app.UseHandler(APIRouter)
	app.UseHandler(ViewRouter)
	// Start app at 3000
	app.Listen(":3000")
}
