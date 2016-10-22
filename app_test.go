package gear_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/teambition/gear"
)

func TestGearAppHello(t *testing.T) {
	app := gear.New()
	app.Use(func(ctx gear.Context) error {
		ctx.End(200, []byte("<h1>Hello!</h1>"))
		return nil
	})
	srv := app.StartBG("")
	defer srv.Close()

	url := "http://" + srv.Addr().String()
	res, err := http.Get(url)
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, 200)

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	require.Nil(t, err)
	require.Equal(t, body, []byte("<h1>Hello!</h1>"))
}
