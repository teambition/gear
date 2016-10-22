package gear_test

import (
	"net/http"
	"testing"

	"github.com/teambition/gear"
)

func BenchmarkGearAppHello(b *testing.B) {
	app := gear.New()
	app.Use(func(ctx gear.Context) error {
		ctx.End(200, []byte("<h1>Hello!</h1>"))
		return nil
	})
	srv := app.StartBG("")
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, _ := http.Get(url)
	res.Body.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if res, err := http.Get(url); err == nil {
			res.Body.Close()
		}
	}
	// 2016-10-22: 5000	    382967 ns/op
}
