package proxy

import (
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
)

type ServiceProxy struct {
	Targets []string
	index   uint64
	client  *http.Client
}

func NewServiceProxy(targets []string) *ServiceProxy {
	if len(targets) == 0 {
		// avoid nil slice
		targets = []string{"http://localhost:9999"}
	}
	return &ServiceProxy{
		Targets: targets,
		client:  &http.Client{},
	}
}

func (p *ServiceProxy) next() string {
	if len(p.Targets) == 1 {
		return p.Targets[0]
	}
	i := atomic.AddUint64(&p.index, 1)
	return p.Targets[int(i)%len(p.Targets)]
}

// Forward forwards the incoming *http.Request to the selected target and returns response.
func (p *ServiceProxy) Forward(origReq *http.Request) (*http.Response, error) {
	target := p.next()
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	// build proxied URL: keep original path + query, but replace scheme/host
	outURL := *origReq.URL
	outURL.Scheme = u.Scheme
	if outURL.Scheme == "" {
		outURL.Scheme = u.Scheme
	}
	outURL.Host = u.Host

	req, err := http.NewRequest(origReq.Method, outURL.String(), origReq.Body)
	if err != nil {
		return nil, err
	}
	// copy headers
	for k, vv := range origReq.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	// forward
	return p.client.Do(req)
}

// CopyResponse copies resp to fiber response writer - helper for routes/proxy handler.
func CopyResponseToFiber(resp *http.Response, w http.ResponseWriter) error {
	// copy headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, err := io.Copy(w, resp.Body)
	return err
}
