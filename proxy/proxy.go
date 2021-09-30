package proxy

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type WebHookURL string

type Proxy struct {
	l              *zap.Logger
	cache          *cache
	backendURL     string
	chanRequestJob chan requestJob
	chanFlushJob   chan requestFlush
}

func NewProxy(ctx context.Context, l *zap.Logger, backendURL string, webHooks []WebHookURL) *Proxy {
	chanRequest := make(chan requestJob)
	chanFlush := make(chan requestFlush)
	c := &cache{
		cacheMap: cacheMap{},
		webHooks: webHooks,
		l:        l,
	}
	go getLoop(ctx, l, backendURL, c, chanRequest, chanFlush)
	return &Proxy{
		l:              l,
		cache:          c,
		backendURL:     backendURL,
		chanRequestJob: chanRequest,
		chanFlushJob:   chanFlush,
	}
}

func getLoop(
	ctx context.Context,
	l *zap.Logger,
	backendURL string,
	c *cache,
	chanRequestJob chan requestJob,
	chanFlush chan requestFlush,
) {
	pendingRequests := map[cacheID][]chan requestJobDone{}
	chanJobDone := make(chan requestJobDone)
	jobRunner := getJobRunner(c, backendURL, chanJobDone)
	for {
		select {
		case command := <-chanFlush:
			c.flush()
			l.Info("cache flush command coming in", zap.String("flushCommand", string(command)))
			c.callWebHooks()
		case nextJob := <-chanRequestJob:
			id := getCacheIDForRequest(nextJob.request)
			pendingRequests[id] = append(pendingRequests[id], nextJob.chanDone)
			requests := pendingRequests[id]
			if len(requests) == 1 {
				l.Info("starting jobrunner for", zap.String("uri", nextJob.request.RequestURI), zap.String("id", string(id)))
				go jobRunner(nextJob, id)
			}
		case jobDone := <-chanJobDone:
			l.Info("request complete", zap.String("id", string(jobDone.id)), zap.Int("num-waiting-clients", len(pendingRequests[jobDone.id])))
			for _, chanPending := range pendingRequests[jobDone.id] {
				chanPending <- jobDone
			}
			delete(pendingRequests, jobDone.id)
		case <-ctx.Done():
			return
		}
	}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/flush":
		command := requestFlush("doit")
		p.chanFlushJob <- command
		return
	}

	switch r.Method {
	case http.MethodGet:
		p.l.Info("serve get request", zap.String("url", r.RequestURI))
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
				http.Error(w, "cache / job error", http.StatusInternalServerError)
				return
			}
			cachedResponse = jobDone.cachedResponse
			p.l.Info("serve response after cache creation", zap.String("url", r.RequestURI))
		} else {
			p.l.Info("serve response from cache", zap.String("url", r.RequestURI))
		}
		for key, values := range cachedResponse.header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}
		_, err := w.Write(cachedResponse.response)
		if err != nil {
			p.l.Info("writing cached response failed", zap.String("url", r.RequestURI))
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
