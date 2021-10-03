package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/foomo/contentfulproxy/proxy"
	"github.com/foomo/keel"
	"github.com/foomo/keel/log"
	"github.com/foomo/keel/net/http/middleware"
)

const (
	ServiceName = "Contentful Proxy"
)

func main() {

	svr := keel.NewServer(
		keel.WithHTTPZapService(true),
		keel.WithHTTPViperService(true),
		keel.WithHTTPPrometheusService(true),
	)

	// get the logger
	l := svr.Logger()

	// register Closers for graceful shutdowns
	svr.AddClosers()

	// define and process flags and arguments
	webserverAddress := flag.String("webserver-address", ":80", "address to bind web server host:port")
	webserverPath := flag.String("webserver-path", "", "path to export the webserver on")
	backendURL := flag.String("backend-url", "https://cdn.contentful.com", "endpoint of the contentful api")
	flag.Parse()
	webhooks, err := getWebhooks()
	if err != nil {
		l.Fatal(err.Error())
	}

	// create proxy
	proxy, _ := proxy.NewProxy(
		context.Background(),
		l,
		*backendURL,
		*webserverPath,
		webhooks,
	)

	// add the service to keel
	svr.AddServices(
		keel.NewServiceHTTP(
			log.WithServiceName(l, ServiceName),
			ServiceName,
			*webserverAddress,
			proxy,
			getMiddleWares()...,
		),
	)
	svr.Run()
}

func getMiddleWares() []middleware.Middleware {
	return []middleware.Middleware{
		middleware.Logger(),
		middleware.Telemetry(),
		middleware.RequestID(),
		middleware.Recover(),
	}
}

func getWebhooks() (proxy.WebHooks, error) {
	args := flag.Args()
	if len(args) == 0 {
		return nil, fmt.Errorf("missing webhook arguments on startup")
	}
	webhooks := proxy.WebHooks{}
	for _, v := range args {
		webhooks = append(webhooks, proxy.WebHookURL(v))
	}
	return webhooks, nil
}
