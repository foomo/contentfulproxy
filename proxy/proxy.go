package proxy

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type Proxy struct {
	cache          *cache
	backendURL     string
	chanRequestJob chan requestJob
	l              *zap.Logger
}

func NewProxy(ctx context.Context, l *zap.Logger, backendURL string) *Proxy {
	chanRequest := make(chan requestJob)
	c := &cache{
		cacheMap: cacheMap{},
	}
	go getLoop(ctx, l, backendURL, c, chanRequest)
	return &Proxy{
		l:              l,
		cache:          c,
		backendURL:     backendURL,
		chanRequestJob: chanRequest,
	}
}

func getLoop(ctx context.Context, l *zap.Logger, backendURL string, c *cache, chanRequestJob chan requestJob) {
	pendingRequests := map[cacheID][]chan requestJobDone{}
	chanJobDone := make(chan requestJobDone)
	jobRunner := getJobRunner(c, backendURL, chanJobDone)
	for {
		select {
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
	switch r.Method {
	case http.MethodGet:
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
		}
		for key, values := range cachedResponse.header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}
		w.Write(cachedResponse.response)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
