package proxy

import "net/http"

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

func getJobRunner(c *cache, backendURL string, chanJobDone chan requestJobDone) jobRunner {
	return func(job requestJob, id cacheID) {
		resp, err := http.Get(backendURL + job.request.URL.RequestURI())
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
