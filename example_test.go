package gear_test

import (
	"fmt"
	"os"
	"time"

	"github.com/teambition/gear"
	"github.com/teambition/gear/middleware"
	"github.com/teambition/gear/middleware/logger"
)

func Example() {
	// Create app
	app := gear.New()

	// Use a default logger middleware
	log := &logger.DefaultLogger{Writer: os.Stdout}
	app.Use(logger.NewLogger(log))

	// Add a static middleware
	// http://localhost:3000/middleware/static.go
	app.Use(middleware.NewStatic(middleware.StaticOptions{
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
			return &gear.Error{Code: 400, Msg: "Invalid view"}
		}
		return ctx.HTML(200, "View: "+view)
	})
	// "http://localhost:3000/abc"
	// "http://localhost:3000/abc/efg"
	ViewRouter.Get("/:others*", func(ctx *gear.Context) error {
		others := ctx.Param("others")
		if others == "" {
			return &gear.Error{Code: 400, Msg: "Invalid path"}
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
			return &gear.Error{Code: 400, Msg: "Invalid user id"}
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

func ExampleBackgroundAPP() {
	app := gear.New()

	app.Use(func(ctx *gear.Context) error {
		return ctx.End(200, []byte("<h1>Hello!</h1>"))
	})

	s := app.Start() // Start at random addr.
	fmt.Printf("App start at: %s\n", s.Addr())
	go func() {
		time.Sleep(time.Second * 3) // Close it after 3 sec
		fmt.Printf("App closed: %s\n", s.Close())
	}()
	s.Wait()
}
