package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"go.uber.org/zap"

	"github.com/foomo/contentfulproxy/proxy"
)

func main() {
	flagAddr := flag.String("addr", ":8888", "address to listen to")
	flag.Parse()
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatal("could not initialize zap logger", err)
	}
	defer l.Sync()
	args := flag.Args()
	if len(args) != 1 {
		l.Error("unexpected number of args - must be exactly one for backendURL")
	}
	p := proxy.NewProxy(
		context.Background(),
		l,
		args[0],
		[]proxy.WebHookURL{
			"https://www.bestbytes.com",
			"https://www.spiegel.de",
		},
	)
	l.Info("starting proxy for", zap.String("backendURL", args[0]), zap.String("addr", *flagAddr))
	l.Error("http listen failed", zap.Error(http.ListenAndServe(*flagAddr, p)))
}
