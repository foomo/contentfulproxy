package proxy

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	responseFoo = `i am a foo response`
	responseBar = `i am bar`
)

type getStats func(path string) int

func GetBackend() (getStats, http.HandlerFunc) {
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

func getTestServer(t *testing.T) (gs func(path string) int, s *httptest.Server) {
	gs, backendHandler := GetBackend()

	p := NewProxy(context.Background(), httptest.NewServer(backendHandler).URL)
	s = httptest.NewServer(p)
	t.Log("we have a proxy in front of it running on", s.URL)
	return gs, s

}

func TestProxy(t *testing.T) {
	gs, server := getTestServer(t)

	get := func(path string) string {
		resp, err := http.Get(server.URL + "/foo")
		assert.NoError(t, err)
		defer resp.Body.Close()
		responseBytes, err := ioutil.ReadAll(resp.Body)
		assert.NoError(t, err)
		return string(responseBytes)
	}
	for j := 0; j < 10; j++ {
		wg := sync.WaitGroup{}
		for i := 0; i < 128; i++ {
			wg.Add(1)
			go func() {
				get("/foo")
				wg.Done()
			}()
		}
		wg.Wait()
	}
	assert.Equal(t, 1, gs("/foo"))

}
