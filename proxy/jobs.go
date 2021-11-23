package proxy

import (
	"github.com/foomo/contentfulproxy/packages/go/log"
	"go.uber.org/zap"
	"net/http"
)

type requestJobDone struct {
	cachedResponse *cachedResponse
	err            error
	id             cacheID
}

type requestJob struct {
	request  *http.Request
	chanDone chan requestJobDone
}

type jobRunner func(job requestJob, id cacheID)

func getJobRunner(l *zap.Logger, c *Cache, backendURL func() string, pathPrefix func() string, chanJobDone chan requestJobDone) jobRunner {
	return func(job requestJob, id cacheID) {
		// backend url is the contentful api domain like https://cdn.contenful.com
		calledURL :=  backendURL() + stripPrefixFromUrl(job.request.URL.RequestURI(), pathPrefix)
		l.Info("URL called by job-runner", log.FURL(calledURL))
		req, err := http.NewRequest("GET", calledURL, nil)
		if err != nil {
			chanJobDone <- requestJobDone{
				id:  id,
				err: err,
			}
			return
		}
		for k, v := range job.request.Header {
			req.Header.Set(k, v[0])
		}
		client := http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			chanJobDone <- requestJobDone{
				id:  id,
				err: err,
			}
			return
		}
		cachedResponse, err := c.set(id, resp)
		if err != nil {
			chanJobDone <- requestJobDone{
				id:  id,
				err: err,
			}
			return
		}
		chanJobDone <- requestJobDone{
			id:             id,
			cachedResponse: cachedResponse,
		}
	}
}
