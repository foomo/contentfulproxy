package proxy

import (
	"context"
	"encoding/json"
	"github.com/foomo/contentfulproxy/packages/go/metrics"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/foomo/contentfulproxy/packages/go/log"
	"go.uber.org/zap"
)

type Info struct {
	WebHooks    []string `json:"webhooks,omitempty"`
	CacheLength int      `json:"cachelength,omitempty"`
	BackendURL  string   `json:"backendurl,omitempty"`
}

type Metrics struct {
	NumUpdate       prometheus.Counter
	NumProxyRequest prometheus.Counter
	NumApiRequest   prometheus.Counter
}

type Proxy struct {
	l              *zap.Logger
	cache          *Cache
	backendURL     func() string
	pathPrefix     func() string
	chanRequestJob chan requestJob
	chanFlushJob   chan requestFlush
	metrics        *Metrics
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case p.pathPrefix() + "/update":
		p.metrics.NumUpdate.Inc()
		command := requestFlush("doit")
		p.chanFlushJob <- command
		return
	case p.pathPrefix() + "/info":
		info := Info{
			WebHooks:    p.cache.webHooks(),
			BackendURL:  p.backendURL(),
			CacheLength: len(p.cache.cacheMap),
		}
		jsonResponse(w, info, http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		p.l.Info("serve get request", zap.String("url", r.RequestURI))
		p.metrics.NumProxyRequest.Inc()
		cacheID := getCacheIDForRequest(r)
		cachedResponse, ok := p.cache.get(cacheID)
		if !ok {
			chanDone := make(chan requestJobDone)
			p.chanRequestJob <- requestJob{
				request:  r,
				chanDone: chanDone,
			}
			jobDone := <-chanDone
			if jobDone.err != nil {
				p.l.Error("Cache / job error", zap.String("url", r.RequestURI))
				http.Error(w, "Cache / job error", http.StatusInternalServerError)
				return
			}
			cachedResponse = jobDone.cachedResponse
			p.l.Info("serve response after cache creation", log.FURL(r.RequestURI), log.FCacheID(string(cacheID)))
			p.metrics.NumApiRequest.Inc()
		} else {
			p.l.Info("serve response from cache", log.FURL(r.RequestURI), log.FCacheID(string(cacheID)))
		}
		for key, values := range cachedResponse.header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}
		_, err := w.Write(cachedResponse.response)
		if err != nil {
			p.l.Info("writing cached response failed", log.FURL(r.RequestURI), log.FCacheID(string(cacheID)))
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func NewProxy(ctx context.Context, l *zap.Logger, backendURL func() string, pathPrefix func() string, webHooks func() []string) (*Proxy, error) {
	chanRequest := make(chan requestJob)
	chanFlush := make(chan requestFlush)
	c := NewCache(l, webHooks)
	go getLoop(ctx, l, backendURL, c, chanRequest, chanFlush)
	return &Proxy{
		l:              l,
		cache:          c,
		backendURL:     backendURL,
		pathPrefix:     pathPrefix,
		chanRequestJob: chanRequest,
		chanFlushJob:   chanFlush,
		metrics:        getMetrics(),
	}, nil
}

func getLoop(
	ctx context.Context,
	l *zap.Logger,
	backendURL func() string,
	c *Cache,
	chanRequestJob chan requestJob,
	chanFlush chan requestFlush,
) {
	pendingRequests := map[cacheID][]chan requestJobDone{}
	chanJobDone := make(chan requestJobDone)
	jobRunner := getJobRunner(c, backendURL, chanJobDone)
	for {
		select {
		case <-chanFlush:
			l.Info("Cache update command coming in")
			c.update()
			c.callWebHooks()
		case nextJob := <-chanRequestJob:
			cacheID := getCacheIDForRequest(nextJob.request)
			pendingRequests[cacheID] = append(pendingRequests[cacheID], nextJob.chanDone)
			requests := pendingRequests[cacheID]
			if len(requests) == 1 {
				l.Info("starting jobrunner for", log.FURL(nextJob.request.RequestURI), log.FCacheID(string(cacheID)))
				go jobRunner(nextJob, cacheID)
			}
		case jobDone := <-chanJobDone:
			l.Info("request complete", log.FCacheID(string(jobDone.id)), log.FNumberOfWaitingClients(len(pendingRequests[jobDone.id])))
			for _, chanPending := range pendingRequests[jobDone.id] {
				chanPending <- jobDone
			}
			delete(pendingRequests, jobDone.id)
		case <-ctx.Done():
			return
		}
	}
}

func jsonResponse(w http.ResponseWriter, v interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "	")
	if statusCode > 0 {
		w.WriteHeader(statusCode)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	err := encoder.Encode(v)
	if err != nil {
		http.Error(w, "could not marshal info export", http.StatusInternalServerError)
	}
}
func getMetrics() *Metrics {
	return &Metrics{
		NumUpdate: metrics.NewCounter("numupdates", "number of times the update webhook was called"),
		NumApiRequest: metrics.NewCounter("numapirequests", "number of times the proxy performed a contentful api-request"),
		NumProxyRequest: metrics.NewCounter("numproxyrequests", "number of times the proxy received an api-request"),
	}
}
