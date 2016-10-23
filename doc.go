// Package gear implements a web framework with context.Context for Go. It focuses on performance and composition.

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
			Root:        "./dist",
			Prefix:      "/static",
			StripPrefix: true,
		}))

		// Add some middleware to app
		app.Use(func(ctx *gear.Context) (err error) {
			// fmt.Println(ctx.IP(), ctx.Method, ctx.Path
			// Do something...

			// Add after hook to the ctx
			ctx.After(func(ctx *gear.Context) {
				// Do something in after hook
				fmt.Println("After hook")
			})
			return
		})

		// Create views router
		ViewRouter := gear.NewRouter("", true)
		// "http://localhost:3000"
		ViewRouter.Get("/", func(ctx *gear.Context) error {
			return ctx.HTML(200, "<h1>Hello, Gear!</h1>")
		})
		// "http://localhost:3000/view/abc"
		// "http://localhost:3000/view/123"
		ViewRouter.Get("/view/:view", func(ctx *gear.Context) error {
			view := ctx.Param("view")
			if view == "" {
				ctx.Status(400)
				return errors.New("Invalid view")
			}
			return ctx.HTML(200, "View: "+view)
		})
		// "http://localhost:3000/abc"
		// "http://localhost:3000/abc/efg"
		ViewRouter.Get("/:others*", func(ctx *gear.Context) error {
			others := ctx.Param("others")
			if others == "" {
				ctx.Status(400)
				return errors.New("Invalid path")
			}
			return ctx.HTML(200, "Request path: /"+others)
		})

		// Create API router
		APIRouter := gear.NewRouter("/api", true)
		// "http://localhost:3000/api/user/abc"
		// "http://localhost:3000/abc/user/123"
		APIRouter.Get("/user/:id", func(ctx *gear.Context) error {
			id := ctx.Param("id")
			if id == "" {
				ctx.Status(400)
				return errors.New("Invalid user id")
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
