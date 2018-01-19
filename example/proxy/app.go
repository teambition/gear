package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pelletier/go-toml"
	"github.com/teambition/gear"
	"github.com/teambition/gear/logging"
	"github.com/vulcand/oxy/forward"
	"github.com/vulcand/oxy/roundrobin"
)

var (
	defaultHandler http.Handler
	confFile       = flag.String("conf", "./config.toml", `config file path.`)
)

// Conf -
type Conf struct {
	Port     int    `toml:"port"`
	CertFile string `toml:"cert_file"`
	KeyFile  string `toml:"key_file"`
	Upstream []struct {
		Host    string   `toml:"host"`
		Servers []string `toml:"servers"`
	} `toml:"upstream"`
}

func main() {
	flag.Parse()
	conf, err := loadConf(*confFile)
	if err != nil {
		logging.Panic(err)
	}
	forwards := loadForwards(conf)

	app := gear.New()
	app.Use(func(ctx *gear.Context) error {
		// we can do some thing here, such as updating cookie
		if lb, ok := forwards[ctx.Host]; ok {
			lb.ServeHTTP(ctx.Res, ctx.Req)
		} else if defaultHandler != nil {
			defaultHandler.ServeHTTP(ctx.Res, ctx.Req)
		} else {
			ctx.HTML(200, "<h1>Gear Proxy</h1>")
		}
		return nil
	})

	port := fmt.Sprintf(":%d", conf.Port)
	if conf.CertFile != "" && conf.KeyFile != "" {
		logging.Panic(app.ListenTLS(port, conf.CertFile, conf.KeyFile))
	} else {
		logging.Panic(app.Listen(port))
	}
}

func loadConf(confPath string) (conf Conf, err error) {
	conf = Conf{}
	if config, err := toml.LoadFile(confPath); err == nil {
		err = config.Unmarshal(&conf)
	}
	return
}

func loadForwards(conf Conf) map[string]http.Handler {
	forwardMap := make(map[string]http.Handler)

	for _, upstream := range conf.Upstream {
		fwd, err := forward.New(forward.PassHostHeader(false), forward.Stream(true))
		if err != nil {
			logging.Panic(err)
		}

		lb, err := roundrobin.New(fwd)
		if err != nil {
			logging.Panic(err)
		}

		for _, srv := range upstream.Servers {
			urlObj, err := url.Parse(srv)
			if err != nil {
				logging.Printf("invalid server %s for %s\n", srv, upstream.Host)
				continue
			}
			lb.UpsertServer(urlObj)
		}
		if len(lb.Servers()) == 0 {
			logging.Printf("no server for %s\n", upstream.Host)
			continue
		}

		if upstream.Host == "*" {
			defaultHandler = lb
		} else {
			forwardMap[upstream.Host] = lb
		}
	}
	return forwardMap
}
