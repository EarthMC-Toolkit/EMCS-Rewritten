package oapi

import "time"

const RATE_LIMIT = 180  // Amt req/min
const QUERY_LIMIT = 100 // Amt identifiers in single req/query

// The global dispatcher for queueing requests while adhering to RATE_LIMIT.
// Stores its own internal token bucket that is automatically refilled at RATE_LIMIT / 60.
//
// Any pending requests must wait for a token to be acquired before executing.
// Initializing the dispatcher with a pre-filled high capacity bucket may cause issues!
var Dispatcher *RequestDispatcher

func init() {
	Dispatcher = NewRequestDispatcher(NewRequestBucket(RATE_LIMIT))
}

type Request func()
type Token struct{}

// A bucket of "tokens" where each token can allow a request to be sent.
// The bucket is responsible for refilling its tokens until the max amount (RATE_LIMIT).
type RequestBucket struct {
	tokens chan Token
}

func NewRequestBucket(reqPerMin int) (bucket *RequestBucket) {
	reqPerMin = min(RATE_LIMIT, reqPerMin)
	reqPerSec := reqPerMin / 60 // Eg: 180req/m becomes 3req/s

	bucket = &RequestBucket{
		tokens: make(chan Token, reqPerSec),
	}

	// Pre-fill
	for range reqPerSec {
		bucket.tokens <- Token{}
	}

	// Token refill rate. Eg: 3req/s becomes 333ms between ticks.
	refillRate := time.Second / time.Duration(reqPerSec)
	ticker := time.NewTicker(refillRate + (time.Millisecond * 10)) // Extra 10ms for safety.

	go func() {
		for range ticker.C {
			select {
			case bucket.tokens <- Token{}:
			default: // full, skip adding token.
			}
		}
	}()

	return
}

// Blocks the goroutine that AcquireToken was called in until a token/request is available.
func (b *RequestBucket) AcquireToken() {
	<-b.tokens
}

// Attempts to a acquire without blocking.
func (b *RequestBucket) TryAcquireToken() bool {
	select {
	case <-b.tokens:
		return true
	default:
		return false
	}
}

// A dispatcher is responsible for queuing and sending requests.
// It does this by waiting for a "token" from its internal bucket.
// If a token is available, a request is immediately sent.
type RequestDispatcher struct {
	reqBucket *RequestBucket
}

func NewRequestDispatcher(bucket *RequestBucket) (dispatcher *RequestDispatcher) {
	dispatcher = &RequestDispatcher{
		reqBucket: bucket,
	}

	return
}

// Submit a request to the queue which will be executed in a goroutine when a token is available.
func (d *RequestDispatcher) Queue(req Request) {
	go func() {
		d.reqBucket.AcquireToken() // Blocks until we have a token to use.
		req()                      // Consume token and run request.
	}()
}

// Burst executes requests as fast as this dispatcher's bucket allows, skipping the queue.
//
// If too little tokens were available to complete all requests with the initial burst, Queue() is called for every remaining request.
func (d *RequestDispatcher) Burst(reqs []Request) {
	for _, req := range reqs {
		select {
		case <-d.reqBucket.tokens:
			req()
		default:
			// No more tokens available, queue remaining requests
			go func(r Request) { d.Queue(r) }(req)
		}
	}
}

func (d *RequestDispatcher) GetBucketTokens() chan Token {
	return d.reqBucket.tokens
}
