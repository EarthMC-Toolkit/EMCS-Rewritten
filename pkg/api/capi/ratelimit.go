package capi

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type RateLimit struct {
	perEndpoint bool
	maxBurst    int // Requests are allowed in bursts up until this amount.

	clients map[string]*rate.Limiter
	mu      sync.Mutex
}

func NewRateLimit(perEndpoint bool, maxBurst uint8) *RateLimit {
	return &RateLimit{
		perEndpoint: perEndpoint,
		maxBurst:    int(maxBurst),
		clients:     make(map[string]*rate.Limiter),
	}
}

func (r *RateLimit) clientLimiter(req *http.Request, rpm int) *rate.Limiter {
	key := getClientIP(req)
	if r.perEndpoint {
		endpoint := req.URL.Path
		key += "@" + endpoint
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.clients[key]
	if !exists {
		interval := rate.Every(time.Minute / time.Duration(rpm)) // interval per request
		limiter = rate.NewLimiter(interval, r.maxBurst)
		r.clients[key] = limiter
	}

	return limiter
}
