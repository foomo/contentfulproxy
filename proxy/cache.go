package proxy

import (
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
)

type cacheID string

type requestFlush string

type cachedResponse struct {
	header   http.Header
	response []byte
}
type cacheMap map[cacheID]*cachedResponse

type cache struct {
	sync.RWMutex
	cacheMap cacheMap
	webHooks []WebHookURL
	l        *zap.Logger
}

func (c *cache) set(id cacheID, response *http.Response) (*cachedResponse, error) {
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	err = response.Body.Close()
	if err != nil {
		return nil, err
	}
	c.Lock()
	defer c.Unlock()
	cr := &cachedResponse{
		header:   response.Header,
		response: responseBytes,
	}
	c.cacheMap[id] = cr
	return cr, nil
}

func (c *cache) get(id cacheID) (*cachedResponse, bool) {
	c.RLock()
	defer c.RUnlock()
	response, ok := c.cacheMap[id]
	return response, ok
}

func (c *cache) flush() {
	c.RLock()
	defer c.RUnlock()
	c.cacheMap = cacheMap{}
}

func (c *cache) callWebHooks() {
	for _, url := range c.webHooks {
		c.l.Info("call webhook", zap.String("url", string(url)))
		_, err := http.Get(string(url))
		if err != nil {
			c.l.Error("could not call webhook", zap.String("url", string(url)))
		}
	}
}

func getCacheIDForRequest(r *http.Request) cacheID {
	id := r.URL.RequestURI()
	keys := make([]string, len(r.Header))
	i := 0
	for k := range r.Header {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		id += k + strings.Join(r.Header[k], "-")
	}
	// hash it here maybe, to keep it shorter
	return cacheID(id)
}
