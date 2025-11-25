package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/fathima-sithara/api-gateway/internal/config"
	"github.com/fathima-sithara/api-gateway/internal/discovery"
	"github.com/sony/gobreaker"
)

// Proxy is the gateway reverse proxy.
type Proxy struct {
	disc discovery.Discovery
	cb   *gobreaker.CircuitBreaker
	log  *zap.Logger
	cfg  config.CircuitBreakerConfig
}

func NewProxy(d discovery.Discovery, logger *zap.Logger, cbCfg config.CircuitBreakerConfig) *Proxy {
	st := gobreaker.Settings{
		Name:        "gateway",
		MaxRequests: 1,
		Interval:    time.Duration(cbCfg.IntervalSec) * time.Second,
		Timeout:     time.Duration(cbCfg.TimeoutSec) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= cbCfg.MaxFailures
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("circuit breaker state", zap.String("name", name), zap.String("from", from.String()), zap.String("to", to.String()))
		},
	}
	cb := gobreaker.NewCircuitBreaker(st)
	return &Proxy{
		disc: d,
		cb:   cb,
		log:  logger,
		cfg:  cbCfg,
	}
}

// ServeReverseProxy proxies incoming request to the service mapped by serviceName.
// pathPrefix is removed before forwarding. e.g. /user/profile -> /profile at user service.
func (p *Proxy) ServeReverseProxy(cctx context.Context, serviceName string, pathPrefix string, w http.ResponseWriter, r *http.Request) {
	// resolve upstream
	upstream, err := p.disc.Lookup(serviceName)
	if err != nil {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
		return
	}

	targetURL, err := url.Parse(upstream)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusBadGateway)
		return
	}

	// create reverse proxy
	director := func(req *http.Request) {
		// rewrite path: trim prefix
		outPath := r.URL.Path
		if pathPrefix != "" && strings.HasPrefix(outPath, pathPrefix) {
			outPath = strings.TrimPrefix(outPath, pathPrefix)
			if outPath == "" {
				outPath = "/"
			}
		}
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = singleJoiningSlash(targetURL.Path, outPath)
		req.URL.RawQuery = r.URL.RawQuery
		req.Header = r.Header.Clone()
		// preserve original remote addr as header
		req.Header.Set("X-Forwarded-For", r.RemoteAddr)
		req.Host = targetURL.Host
	}

	proxy := &httputil.ReverseProxy{
		Director:  director,
		Transport: p.breakerTransport(http.DefaultTransport),
		ModifyResponse: func(resp *http.Response) error {
			// optionally sanitize or add headers
			resp.Header.Set("X-Gateway", "api-gateway")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			p.log.Error("proxy error", zap.Error(err))
			http.Error(w, "upstream error", http.StatusBadGateway)
		},
	}

	// Execute proxy.ServeHTTP through circuit breaker
	_, err = p.cb.Execute(func() (interface{}, error) {
		// ServeHTTP may block; call in goroutine or direct
		proxy.ServeHTTP(w, r)
		return nil, nil
	})
	if err != nil {
		// translate circuit breaker error
		p.log.Error("request blocked by circuit breaker", zap.Error(err))
		http.Error(w, "service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
}

func (p *Proxy) breakerTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return roundTripperWithBreaker{
		next: next,
		cb:   p.cb,
		log:  p.log,
	}
}

type roundTripperWithBreaker struct {
	next http.RoundTripper
	cb   *gobreaker.CircuitBreaker
	log  *zap.Logger
}

func (rt roundTripperWithBreaker) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := rt.cb.Execute(func() (interface{}, error) {
		ctx, cancel := context.WithTimeout(req.Context(), 30*time.Second)
		defer cancel()
		req = req.Clone(ctx)
		resp, err := rt.next.RoundTrip(req)
		if err != nil {
			// network or transport error
			return nil, err
		}
		// handle 5xx as failure to increase CB counts
		if resp.StatusCode >= 500 {
			// consume body to allow reuse
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("upstream status %d", resp.StatusCode)
		}
		return resp, nil
	})
	if err != nil {
		rt.log.Error("breaker transport error", zap.Error(err))
		return nil, err
	}
	// success: type assert
	if r, ok := res.(*http.Response); ok {
		return r, nil
	}
	return nil, errors.New("invalid roundtrip result")
}

// helper to join paths
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
