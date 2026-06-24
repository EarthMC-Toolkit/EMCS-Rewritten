package capi

import (
	"bytes"
	"emcsrw/pkg/utils/logutil"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

// Header fields specified by RFC 7230 that aren't supposed to be forwaded.
var hopByHop = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

var noRedirectClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("too many redirects. sussy baka")
		}

		logutil.Println(logutil.BLUE, "allowed redirect to: ", req.URL.String())
		return nil
	},
}

// A simple CORS reverse proxy, where only a whitelist of hosts are allowed.
type Proxy struct {
	allowedHosts []string
	rl           *RateLimit
	rpm          int
}

func NewProxy(rl *RateLimit, reqPerMin uint8, allowedHosts []string) *Proxy {
	return &Proxy{rl: rl, rpm: int(reqPerMin), allowedHosts: allowedHosts}
}

// Handles CORS preflight, parses the target URL and forwards the request to the upstream HTTPS endpoint.
func (p *Proxy) handler(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
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
	lim := p.rl.clientLimiter(r, p.rpm) // amt of reqs/min allowed per client
	return lim.Allow()
}

func (p *Proxy) isAllowedHost(hostname string) bool {
	hostname = strings.ToLower(hostname)
	return slices.Contains(p.allowedHosts, hostname)
}

// Extracts and validates the target URL from the raw request query string.
func (p *Proxy) getTargetUrl(r *http.Request) (*url.URL, error) {
	targetParam := r.URL.Query().Get("target")
	if targetParam == "" {
		return nil, errors.New("missing target param")
	}
	if strings.Count(strings.ToLower(targetParam), "https://") > 2 {
		return nil, errors.New("target exceeded max 2 embedded urls")
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
	host := u.Hostname()
	if host == "web.archive.org" {
		idx := strings.Index(strings.ToLower(u.Path), "https://")
		if idx == -1 {
			return nil, errors.New("no embedded url")
		}

		innerURL, err := url.Parse(u.Path[idx:])
		if err != nil {
			return nil, errors.New("invalid url given to wayback")
		}
		if innerURL.Host == "" {
			return nil, errors.New("missing host in wayback url")
		}
		if innerURL.Scheme != "https" {
			return nil, errors.New("invalid target. scheme must be https")
		}

		host = innerURL.Hostname()
	}

	// No archive and not allowed host, NONE SHALL PASS
	if !p.isAllowedHost(host) {
		return nil, fmt.Errorf("blocked host \"%s\". target must ultimately point to %s", u.Hostname(), p.allowedHosts[0])
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
	req.Header = sanitizeHeader(r)

	resp, err := noRedirectClient.Do(req)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// MANDATORY. So upstream can handle response headers correctly.
	for k, v := range resp.Header {
		if hk := http.CanonicalHeaderKey(k); hopByHop[hk] {
			continue
		}

		if !strings.HasPrefix(strings.ToLower(k), "access-control-") {
			w.Header().Set(k, v[0])
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func cloneBody(r *http.Request) ([]byte, error) {
	if r.ContentLength <= 0 {
		return nil, nil
	}

	return io.ReadAll(r.Body)
}

// Copies the Header from r and removes browser-specific context and fingerprinting
// fields that are not required for upstream API compatibility or proxy functionality.
//
// It also adds privacy fields to the header that the upstream server can optionally respect.
func sanitizeHeader(r *http.Request) http.Header {
	h := r.Header.Clone() // Copy header fields from the original req

	// Remove fingerprinting and tracking fields
	h.Del("Referer")
	h.Del("Sec-Fetch-Dest")
	h.Del("Sec-Fetch-Mode")
	h.Del("Sec-Fetch-Site")
	h.Del("Sec-Fetch-User")
	h.Del("Sec-Ch-Ua")
	h.Del("Sec-CH-UA")
	h.Del("Sec-Ch-Ua-Mobile")
	h.Del("Sec-CH-UA-Mobile")
	h.Del("Sec-Ch-Ua-Platform")
	h.Del("Sec-CH-UA-Platform")
	h.Del("Sec-Ch-Ua-Platform-Version")
	h.Del("Sec-CH-UA-Platform-Version")
	h.Del("Sec-Ch-Ua-Full-Version-List")
	h.Del("Sec-CH-UA-Full-Version-List")

	// Privacy (bc why not)
	h.Set("DNT", "1")
	h.Set("X-Do-Not-Track", "1")
	h.Set("Sec-GPC", "1")

	return h
}
