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

func getJobRunner(c *Cache, backendURL func() string, chanJobDone chan requestJobDone) jobRunner {
	return func(job requestJob, id cacheID) {
		req, err := http.NewRequest("GET", backendURL()+job.request.URL.RequestURI(), nil)
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
