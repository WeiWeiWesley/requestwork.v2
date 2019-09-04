package requestwork

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

type job struct {
	req     *http.Request
	handler func(resp *http.Response, err error) error

	end chan error
}

type result struct {
	resp *http.Response
	err  error
}

//DefaultMaxIdleConnPerHost max idle
const DefaultMaxIdleConnPerHost = 20

//New return http worker
func New(threads int) *Worker {

	tr := &http.Transport{
		Proxy: NoProxyAllowed,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		MaxIdleConnsPerHost:   threads,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		MaxIdleConns:          threads,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   time.Second * 120,
	}
	w := &Worker{
		jobQuene: make(chan *job),
		threads:  threads,
		client:   client,
	}

	go w.start()
	return w

}

//NoProxyAllowed no proxy
func NoProxyAllowed(request *http.Request) (*url.URL, error) {
	return nil, nil
}

//Worker instance
type Worker struct {
	jobQuene chan *job
	threads  int
	client   *http.Client
}

//Execute exec http request
func (w *Worker) Execute(req *http.Request, h func(resp *http.Response, err error) error) (err error) {

	j := &job{req, h, make(chan error)}
	w.jobQuene <- j
	return <-j.end

}

func (w *Worker) run() {
	for j := range w.jobQuene {
		c := make(chan error, 1)
		go func() {
			c <- j.handler(w.client.Do(j.req))
		}()
		select {
		case <-j.req.Context().Done():

			j.end <- j.req.Context().Err()
		case err := <-c:
			j.end <- err
		}
	}

}

func (w *Worker) start() {

	for i := 0; i < w.threads; i++ {
		go w.run()
	}

}
