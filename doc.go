// Package gear implements a lightweight, composable and high performance web service framework for Go.

/*
Example:

	package main

	import (
		"fmt"
		"time"

		"github.com/teambition/gear"
		"github.com/teambition/gear/logging"
		"github.com/teambition/gear/middleware/static"
	)

	func main() {
		// Create app
		app := gear.New()

		// Use a default logger middleware
		app.UseHandler(logging.Default())

		// Add a static middleware
		// http://localhost:3000/middleware/static.go
		app.Use(static.New(static.Options{
			Root:        "./middleware",
			Prefix:      "/middleware",
			StripPrefix: true,
		}))

		// Add some middleware to app
		app.Use(func(ctx *gear.Context) (err error) {
			// fmt.Println(ctx.IP(), ctx.Method, ctx.Path
			// Do something...

			// Add after hook to the ctx
			ctx.After(func() {
				// Do something in after hook
				fmt.Println("After hook")
			})
			return
		})

		// Create views router
		ViewRouter := gear.NewRouter()
		// "http://localhost:3000"
		ViewRouter.Get("/", func(ctx *gear.Context) error {
			return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
		})
		// "http://localhost:3000/view/abc"
		// "http://localhost:3000/view/123"
		ViewRouter.Get("/view/:view", func(ctx *gear.Context) error {
			view := ctx.Param("view")
			if view == "" {
				return gear.ErrBadRequest.WithMsg("Invalid view")
			}
			return ctx.HTML(200, "View: "+view)
		})
		// "http://localhost:3000/abc"
		// "http://localhost:3000/abc/efg"
		ViewRouter.Get("/:others*", func(ctx *gear.Context) error {
			others := ctx.Param("others")
			if others == "" {
				return gear.ErrBadRequest.WithMsg("Invalid path")
			}
			return ctx.HTML(200, "Request path: /"+others)
		})

		// Create API router
		APIRouter := gear.NewRouter(gear.RouterOptions{Root: "/api", IgnoreCase: true})
		// "http://localhost:3000/api/user/abc"
		// "http://localhost:3000/abc/user/123"
		APIRouter.Get("/user/:id", func(ctx *gear.Context) error {
			id := ctx.Param("id")
			if id == "" {
				return gear.ErrBadRequest.WithMsg("Invalid user id")
			}
			return ctx.JSON(200, map[string]string{
				"Method": ctx.Method,
				"Path":   ctx.Path,
				"UserID": id,
			})
		})

		// Must add APIRouter first.
		app.UseHandler(APIRouter)
		app.UseHandler(ViewRouter)
		// Start app at 3000
		app.Error(app.Listen(":3000"))
	}

Learn more at https://github.com/teambition/gear
*/

package gear

// Version is Gear's version
const Version = "1.9.13"
