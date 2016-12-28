package gear_test

import (
	"net/http"
	"testing"

	"github.com/teambition/gear"
)

// go test -bench=. -run BenchmarkGearAppHello
// 10000	    353521 ns/op	   17469 B/op	     131 allocs/op
//
func BenchmarkGearAppHello(b *testing.B) {
	app := gear.New()
	for i := 0; i < 100; i++ {
		app.Use(func(ctx *gear.Context) error {
			return nil
		})
	}
	app.Use(func(ctx *gear.Context) error {
		return ctx.End(200, []byte("<h1>Hello!</h1>"))
	})
	srv := app.Start()
	defer srv.Close()

	url := "http://" + srv.Addr().String() + "/?foo[bar]=baz"
	b.N = 10000
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		res.Body.Close()
	}
}
