# contentfulproxy
An experimental reverse proxy cache for the contentful API. Point your contentful client to the proxy instead of the contenful API endpoints (cdn.contentful.com or preview.contentful.com) and the proxy will cache the API-responses and return them.

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


# License
Copyright (c) foomo under the LGPL 3.0 license.
