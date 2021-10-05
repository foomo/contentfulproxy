package proxy

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/foomo/contentfulproxy/packages/go/log"
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

type Cache struct {
	sync.RWMutex
	cacheMap cacheMap
	webHooks func() []string
	l        *zap.Logger
}

func (c *Cache) set(id cacheID, response *http.Response) (*cachedResponse, error) {
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

func (c *Cache) get(id cacheID) (*cachedResponse, bool) {
	c.RLock()
	defer c.RUnlock()
	response, ok := c.cacheMap[id]
	return response, ok
}

func (c *Cache) update() {
	c.RLock()
	defer c.RUnlock()
	c.cacheMap = cacheMap{}
	c.l.Info("flushed the cache")
}

func (c *Cache) callWebHooks() {
	for _, url := range c.webHooks() {
		go func(url string, l *zap.Logger) {
			l.Info("call webhook")
			_, err := http.Get(url)
			if err != nil {
				l.Error("error while calling webhook", zap.Error(err))
			}
		}(url, c.l.With(log.FURL(url)))
	}
}

func NewCache(l *zap.Logger, webHooks func() []string) *Cache {
	c := &Cache{
		cacheMap: cacheMap{},
		webHooks: webHooks,
		l:        l.With(log.FServiceRoutine("cache")),
	}
	return c
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
		// each cf request is signed by an uuid in the X-Request-Id header
		// we have to remove this from the ID-creation
		if k != "X-Request-Id" {
			id += k + strings.Join(r.Header[k], "-")
		}
	}
	// hash it here maybe, to keep it shorter
	hash := md5.New()
	hash.Write([]byte(id))
	id = hex.EncodeToString(hash.Sum(nil))
	return cacheID(id)
}
