package capi

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

// Proxy is a simple authenticated CORS reverse proxy.
type Proxy struct {
	secret     string
	authHeader string
}

// Validates auth, handles CORS preflight, parses the target URL and
// forwards the request to the upstream HTTPS endpoint.
func (p *Proxy) handler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(p.authHeader) != p.secret {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodOptions {
		handleOptions(w)
		return
	}

	if u := parse(w, r); u != nil {
		p.forward(w, r, u)
	}
}

// Responds to CORS preflight requests initiated via the OPTIONS method.
func handleOptions(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.WriteHeader(http.StatusNoContent)
}

// Extracts and validates the target URL from the raw request query string.
func parse(w http.ResponseWriter, r *http.Request) *url.URL {
	if r.URL.RawQuery == "" {
		http.Error(w, "missing a target url", http.StatusBadRequest)
		return nil
	}
	u, err := url.Parse(r.URL.RawQuery)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		http.Error(w, "invalid target (https only)", http.StatusBadRequest)
		return nil
	}

	return u
}

// Proxies the request to the validated HTTPS target, buffering the request body
// for safe forwarding and streaming the upstream response back to the client.
func (p *Proxy) forward(w http.ResponseWriter, r *http.Request, targetUrl *url.URL) {
	body, err := p.cloneBody(r)
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
	req.Header.Del(p.authHeader) // prevent leaking secret in upstream req

	resp, err := http.DefaultClient.Do(req)
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

func (p *Proxy) cloneBody(r *http.Request) ([]byte, error) {
	if r.ContentLength <= 0 {
		return nil, nil
	}

	return io.ReadAll(r.Body)
}
