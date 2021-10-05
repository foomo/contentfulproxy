package main

import (
	"context"

	"github.com/foomo/contentfulproxy/packages/go/config"
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
		keel.WithHTTPZapService(false),
		keel.WithHTTPViperService(false),
		keel.WithHTTPPrometheusService(false),
	)

	// get the logger
	l := svr.Logger()

	// register Closers for graceful shutdowns
	svr.AddClosers()

	c := svr.Config()
	webhookURLs := config.DefaultWebhookURLs(c)
	webserverAddress := config.DefaultWebserverAddress(c)
	webserverPath := config.DefaultWebserverPath(c)
	backendURL := config.DefaultBackendURL(c)


	// create proxy
	proxy, _ := proxy.NewProxy(
		context.Background(),
		log.WithServiceName(l, ServiceName),
		backendURL,
		webserverPath,
		webhookURLs,
	)

	// add the service to keel
	svr.AddServices(
		keel.NewServiceHTTP(
			log.WithServiceName(l, ServiceName),
			ServiceName,
			webserverAddress(),
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
