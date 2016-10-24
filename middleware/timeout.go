package middleware

import (
	"time"

	"github.com/teambition/gear"
)

// NewTimeout returns a timeout middleware with time.Duration and timeout hook.
//
//	app := gear.New()
//	app.Use(NewTimeout(time.Second, func(ctx *gear.Context) {
//    // timeout hook
//		ctx.Status(504)
//		ctx.String("Service timeout")
//	}))
//	app.Use(func(ctx *gear.Context) error {
//		// some process maybe timeout...
//		c, _ := ctx.WithTimeout(time.Second * 2)
//		select {
//		case <-ctx.Done(): // this case will always reached
//		case <-c.Done(): // this case maybe reached... but elapsed time should be 1 sec.
//		}
//		return nil
//	})
//	app.Use(func(ctx *gear.Context) error {
//		// if timeout, the rest of middleware will not run.
//		panic("this middleware unreachable")
//	})
func NewTimeout(du time.Duration, h gear.Hook) gear.Middleware {
	return func(ctx *gear.Context) error {
		c, _ := ctx.WithTimeout(du)
		go func() {
			select {
			case <-c.Done():
				ctx.Cancel()
				h(ctx)
			case <-ctx.Done():
			}
		}()
		return nil
	}
}
