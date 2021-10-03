package proxy

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type WebHookURL string
type WebHooks []WebHookURL

type Info struct {
	WebHooks    WebHooks `json:"webhooks,omitempty"`
	CacheLength int      `json:"cachelength,omitempty"`
	BackendURL  string   `json:"backendurl,omitempty"`
}

type Proxy struct {
	l              *zap.Logger
	cache          *cache
	backendURL     string
	pathPrefix     string
	chanRequestJob chan requestJob
	chanFlushJob   chan requestFlush
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case p.pathPrefix + "/update":
		command := requestFlush("doit")
		p.chanFlushJob <- command
		return
	case p.pathPrefix + "/info":
		info := Info{
			WebHooks:    p.cache.webHooks,
			BackendURL:  p.backendURL,
			CacheLength: len(p.cache.cacheMap),
		}
		jsonResponse(w, info, http.StatusOK)
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
				p.l.Error("cache / job error", zap.String("url", r.RequestURI))
				http.Error(w, "cache / job error", http.StatusInternalServerError)
				return
			}
			cachedResponse = jobDone.cachedResponse
			p.l.Info("serve response after cache creation", zap.String("url", r.RequestURI), zap.String("cache id", string(cacheID)))
		} else {
			p.l.Info("serve response from cache", zap.String("url", r.RequestURI), zap.String("cache id", string(cacheID)))
		}
		for key, values := range cachedResponse.header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}
		_, err := w.Write(cachedResponse.response)
		if err != nil {
			p.l.Info("writing cached response failed", zap.String("url", r.RequestURI), zap.String("cache id", string(cacheID)))
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func NewProxy(ctx context.Context, l *zap.Logger, backendURL string, pathPrefix string, webHooks WebHooks) (*Proxy, error) {
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
		pathPrefix:     pathPrefix,
		chanRequestJob: chanRequest,
		chanFlushJob:   chanFlush,
	}, nil
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
			l.Info("cache flush command coming in", zap.String("flushCommand", string(command)))
			c.flush()
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
