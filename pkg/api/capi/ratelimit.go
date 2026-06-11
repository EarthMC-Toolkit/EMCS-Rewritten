package capi

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const BURST = 1

var clients = make(map[string]*rate.Limiter)
var mu sync.Mutex

func getLimiter(ip, endpoint string, rpm int) *rate.Limiter {
	key := ip + "|" + endpoint

	mu.Lock()
	defer mu.Unlock()

	limiter, exists := clients[key]
	if !exists {
		r := rate.Every(time.Minute / time.Duration(rpm)) // interval per request
		limiter = rate.NewLimiter(r, BURST)
		clients[key] = limiter
	}

	return limiter
}
