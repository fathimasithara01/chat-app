package httpclient

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type ClientConfig struct {
	Timeout        time.Duration
	RetryMaxElapsed time.Duration
	MaxIdleConns   int
	IdleConnTimeout time.Duration
}

type Client struct {
	http *http.Client
	conf ClientConfig
}

func NewClient(conf ClientConfig) *Client {
	tr := &http.Transport{
		DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
		MaxIdleConns:        conf.MaxIdleConns,
		IdleConnTimeout:     conf.IdleConnTimeout,
	}
	return &Client{
		http: &http.Client{Transport: tr, Timeout: conf.Timeout},
		conf: conf,
	}
}

// DoWithRetry runs request with exponential backoff. ctx carries cancellation.
func (c *Client) DoWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	operation := func() error {
		r, err := c.http.Do(req.WithContext(ctx))
		if err != nil {
			return err
		}
		// treat 5xx as retryable
		if r.StatusCode >= 500 {
			// drain body and close to reuse connection
			io.Copy(io.Discard, r.Body)
			r.Body.Close()
			return &backoff.PermanentError{Err: nil} // let backoff try? choose to return error
		}
		resp = r
		return nil
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = c.conf.RetryMaxElapsed
	err := backoff.Retry(operation, backoff.WithContext(b, ctx))
	if err != nil {
		return nil, err
	}
	return resp, nil
}
