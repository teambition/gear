package gear_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/teambition/gear"
)

func BenchmarkGearAppHello(b *testing.B) {
	app := gear.New()
	app.Use(func(ctx *gear.Context) error {
		return ctx.End(200, []byte("<h1>Hello!</h1>"))
	})
	srv := app.Start()
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, _ := http.Get(url)
	ioutil.ReadAll(res.Body)
	res.Body.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		ioutil.ReadAll(res.Body)
		res.Body.Close()
	}
	// 2016-10-27: 20000	     79827 ns/op
}
