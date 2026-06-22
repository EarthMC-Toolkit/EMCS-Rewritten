package capi

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// Proxy is a simple authenticated CORS reverse proxy.
type Proxy struct {
	rl          *RateLimit
	allowedHost string
}

func NewProxy(rl *RateLimit, allowedHost string) *Proxy {
	return &Proxy{rl: rl, allowedHost: allowedHost}
}

// Validates auth, handles CORS preflight, parses the target URL and
// forwards the request to the upstream HTTPS endpoint.
func (p *Proxy) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		handleOptions(w)
		return
	}

	turl, err := p.getTargetUrl(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !p.allow(r) {
		http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
		return
	}

	p.forward(w, r, turl)
}

func (p *Proxy) allow(r *http.Request) bool {
	lim := p.rl.clientLimiter(r, 30) // amt of reqs/min allowed per client
	return lim.Allow()
}

func (p *Proxy) isAllowedHost(host string) bool {
	return host == p.allowedHost || strings.HasSuffix(host, "."+p.allowedHost)
}

// Extracts and validates the target URL from the raw request query string.
func (p *Proxy) getTargetUrl(r *http.Request) (*url.URL, error) {
	targetParam := r.URL.Query().Get("target")
	if targetParam == "" {
		return nil, errors.New("missing target param")
	}
	if strings.Count(strings.ToLower(targetParam), "https://") > 2 {
		return nil, errors.New("max 2 urls exceeded. target must ultimately point to " + p.allowedHost + " (wayback allowed)")
	}

	u, err := url.Parse(targetParam)
	if err != nil {
		return nil, errors.New("failed to parse target: " + err.Error())
	}
	if u.Host == "" {
		return nil, errors.New("failed to parse target: empty host")
	}
	if u.Scheme != "https" {
		return nil, errors.New("invalid target. scheme must be https")
	}

	// In case we are retrieving an archive we need to allow that, but perform some extra
	// validations to avoid malicious rogue actors that want to molest our sweet proxy >:(
	if u.Host == "web.archive.org" {
		idx := strings.Index(strings.ToLower(u.Path), "https://")
		if idx == -1 {
			return nil, errors.New("no embedded url")
		}

		u, err = url.Parse(u.Path[idx:])
		if err != nil {
			return nil, errors.New("invalid url given to wayback")
		}
		if u.Scheme == "" {
			return nil, errors.New("missing host in wayback url")
		}
		if u.Scheme != "https" {
			return nil, errors.New("invalid target. scheme must be https")
		}
	}

	// No archive and not allowed host, NONE SHALL PASS
	if !p.isAllowedHost(u.Host) {
		return nil, errors.New("blocked host. target does not ultimately point to " + p.allowedHost)
	}

	return u, nil
}

// Proxies the request to the validated HTTPS target, buffering the request body
// for safe forwarding and streaming the upstream response back to the client.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, targetUrl *url.URL) {
	body, err := cloneBody(r)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest(r.Method, targetUrl.String(), bytes.NewReader(body))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	req.Header = r.Header.Clone()

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Allows CORS access so JS in the browser can read the response.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func cloneBody(r *http.Request) ([]byte, error) {
	if r.ContentLength <= 0 {
		return nil, nil
	}

	return io.ReadAll(r.Body)
}

// Responds to CORS preflight requests initiated via the OPTIONS method.
func handleOptions(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.WriteHeader(http.StatusNoContent)
}
