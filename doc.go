// Package gear implements a web framework with context.Context for Go. It focuses on performance and composition.
// Version v0.1.0

/*
Example:

	package main

	import (
		"errors"
		"fmt"

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

			// Add after hook to the ctx
			ctx.After(func(ctx gear.Context) {
				// Do something in after hook
				fmt.Println("After hook")
			})
			return
		})

		// Create views router
		ViewRouter := gear.NewRouter("", true)
		// "http://localhost:3000"
		ViewRouter.Get("/", func(ctx gear.Context) (err error) {
			ctx.HTML(200, "<h1>Hello, Gear!</h1>")
			return
		})
		// "http://localhost:3000/view/abc"
		// "http://localhost:3000/view/123"
		ViewRouter.Get("/view/:view", func(ctx gear.Context) (err error) {
			if view := ctx.Param("view"); view == "" {
				ctx.Status(400)
				err = errors.New("Invalid view")
			} else {
				ctx.HTML(200, "View: "+view)
			}
			return
		})
		// "http://localhost:3000/abc"
		// "http://localhost:3000/abc/efg"
		ViewRouter.Get("/:others*", func(ctx gear.Context) (err error) {
			if others := ctx.Param("others"); others == "" {
				ctx.Status(400)
				err = errors.New("Invalid path")
			} else {
				ctx.HTML(200, "Request path: /"+others)
			}
			return
		})

		// Create API router
		APIRouter := gear.NewRouter("/api", true)
		// "http://localhost:3000/api/user/abc"
		// "http://localhost:3000/abc/user/123"
		APIRouter.Get("/user/:id", func(ctx gear.Context) (err error) {
			if id := ctx.Param("id"); id == "" {
				ctx.Status(400)
				err = errors.New("Invalid user id")
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

Learn more at https://github.com/teambition/gear
*/

package gear
