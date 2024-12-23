[![Build Status](https://github.com/foomo/contentfulproxy/actions/workflows/test.yml/badge.svg?branch=main&event=push)](https://github.com/foomo/contentfulproxy/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/foomo/contentfulproxy)](https://goreportcard.com/report/github.com/foomo/contentfulproxy)
[![Coverage Status](https://coveralls.io/repos/github/foomo/contentfulproxy/badge.svg?branch=main&)](https://coveralls.io/github/foomo/contentfulproxy?branch=main)
[![GoDoc](https://godoc.org/github.com/foomo/contentfulproxy?status.svg)](https://godoc.org/github.com/foomo/contentfulproxy)

<p align="center">
  <img alt="sesamy" src=".github/assets/contentfulproxy.png"/>
</p>

# contentfulproxy

> An experimental reverse proxy cache for the contentful API. Point your contentful client to the proxy instead of the contenful API endpoints (cdn.contentful.com or preview.contentful.com) and the proxy will cache the API-responses and return them.

## Configuration
```bash
# environment variables
WEBHOOK_URLS        // comma-separated list of URLs called after a cache update
WEBSERVER_PATH      // set this if the service is running in a subdirectory
BACKEND_URL	    // the contentful api you want to use
WEBSERVER_ADDRESS   // address and port the service should listen to
```

## Usages

### Use as library
```go
logger := zap.New(...)
webhookURLs := func()[]string{ return []string{} }
webserverPath := func()string{ return "/proxy/exposed/on/path" }
backendURL := func()string{ return "https://cdn.contentful.com" }

proxy, _ := proxy.NewProxy(
	context.Background(),
	logger,
	backendURL,
	webserverPath,
	webhookURLs,
)

http.ListenAndServe(":80", proxy)
```

### Run in docker
```bash
$ docker run -p 8080:80 -e WEBHOOK_URLS=https://my-service/webhook1,https://my-service/webhook2 foomo/contentfulproxy
```

### Use as service in a squadron
```yaml
version: '1.0'
name: my-squadron
squadron:
	contentfulproxy:
		chart: /path/to/helm/chart
		values:
			image:
				tag: 0.0.1
				repository: foomo/contentfulproxy
			ports:
				- 80
			env:
				WEBHOOK_URLS: https://my-service/webhook1,https://my-service/webhook2
				WEBSERVER_PATH: /proxy/exposed/on/path
				BACKEND_URL: https://cdn.contentful.com
			ingress:
				paths:
					- path: /proxy/exposed/on/path
						port: 80
```

## How to Contribute

Make a pull request...

## License

Distributed under MIT License, please see license file within the code for more details.

_Made with ♥ [foomo](https://www.foomo.org) by [bestbytes](https://www.bestbytes.com)_
