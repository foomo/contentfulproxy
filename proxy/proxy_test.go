package proxy

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const (
	responseFoo = `i am a foo response`
	responseBar = `i am bar response`
	responseUpdate = `update`
	responseFlush = `flush`
)

type getStats func(path string) int

func GetBackend(t *testing.T) (getStats, http.HandlerFunc) {
	stats := map[string]int{}
	statLock := sync.RWMutex{}
	return func(path string) int {
			statLock.RLock()
			defer statLock.RUnlock()
			count, ok := stats[path]
			if !ok {
				return -1
			}
			return count
		}, func(w http.ResponseWriter, r *http.Request) {
			statLock.Lock()
			stats[r.URL.Path]++
			statLock.Unlock()

			t.Log("backend: url called", r.URL.Path)

			switch r.URL.Path {
			case "/foo":
				w.Write([]byte(responseFoo))
				return
			case "/bar":
				w.Write([]byte(responseBar))
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}
}

func GetWebHook(t *testing.T) (getStats, http.HandlerFunc) {
	stats := map[string]int{}
	statLock := sync.RWMutex{}
	return func(path string) int {
			statLock.RLock()
			defer statLock.RUnlock()
			count, ok := stats[path]
			if !ok {
				return -1
			}
			return count
		}, func(w http.ResponseWriter, r *http.Request) {
			statLock.Lock()
			stats[r.URL.Path]++
			statLock.Unlock()

			t.Log("webhook: url called", r.URL.Path)

			switch r.URL.Path {
			case "/update":
				w.Write([]byte(responseUpdate))
				return
			case "/flush":
				w.Write([]byte(responseFlush))
				return
			}
			http.Error(w, "not found", http.StatusNotFound)
		}
}

func getTestServer(t *testing.T) (gs func(path string) int, ws func(path string) int, s *httptest.Server) {

	l, _ := zap.NewProduction()

	gs, backendHandler := GetBackend(t)
	ws, webHookHandler := GetWebHook(t)
	backendServer := httptest.NewServer(backendHandler)
	webHookServer := httptest.NewServer(webHookHandler)

	p, _ := NewProxy(
		context.Background(),
		l,
		backendServer.URL,
		"",
		[]WebHookURL{
			WebHookURL(webHookServer.URL + "/update"),
			WebHookURL(webHookServer.URL + "/flush"),
		},
	)
	s = httptest.NewServer(p)
	t.Log("we have a proxy in front of it running on", s.URL)
	return gs, ws, s

}

func TestProxy(t *testing.T) {
	gs, ws, server := getTestServer(t)

	get := func(path string) string {
		resp, err := http.Get(server.URL + path)
		assert.NoError(t, err)
		defer resp.Body.Close()
		responseBytes, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		return string(responseBytes)
	}
	for j := 0; j < 10; j++ {
		wg := sync.WaitGroup{}
		for i := 0; i < 1; i++ {
			wg.Add(1)
			go func() {
				get("/foo")
				wg.Done()
			}()
		}
		wg.Wait()
	}
	assert.Equal(t, 1, gs("/foo"))

	//
	http.Get(server.URL + "/update")


	//
	assert.Equal(t, 1, ws("/update"))
	assert.Equal(t, 1, ws("/flush"))

}
