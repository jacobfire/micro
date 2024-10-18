package main

import (
	"net/http"
	"time"
)

type RoundTripper interface {
	RoundTrip(r *http.Request) (*http.Response, error)
}

type retryRoundTripper struct {
	next       RoundTripper
	maxRetries int
	delay      time.Duration
}

func (rt *retryRoundTripper) RoundTripp(req *http.Request) (res *http.Response, err error) {
	for attempts := 0; attempts < rt.maxRetries; attempts++ {
		res, err = rt.next.RoundTrip(req)
		if err == nil && res.StatusCode == http.StatusOK {
			break
		}

		<-time.After(rt.delay)
	}

	return
}

/*
type Client struct {
	httpClient *http.Client
}
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &retryRoundTripper{
				next: http.DefaultTransport
				maxRetries: 3,
				delay: 10 * time.Millisecond,
			},
		}
	}
}
*/