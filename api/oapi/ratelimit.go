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

type Request func() error
type RequestNoErr func()
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

// Blocks the goroutine that this func was called in until a token/request is available.
func (b *RequestBucket) AcquireToken() {
	<-b.tokens
}

// Attempts to acquire a token without blocking the goroutine, reporting whether acquisition succeeded.
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
// If a token is available, a request is sent either synchronously or
// asynchronously according to which method was used to queue it.
type RequestDispatcher struct {
	reqBucket *RequestBucket
}

func NewRequestDispatcher(bucket *RequestBucket) (dispatcher *RequestDispatcher) {
	dispatcher = &RequestDispatcher{
		reqBucket: bucket,
	}

	return
}

// Executes the req synchronously after waiting to acquire a token, both of which block the caller.
//
// To make an async request, prefer EnqueueAsync or EnqueueAsyncErr for error logging.
func (d *RequestDispatcher) Enqueue(req Request) error {
	d.reqBucket.AcquireToken() // Blocks until we have a token to use.
	return req()               // Consume token and run a request.
}

// Runs a goroutine that handles executing req once a token is acquired.
// This is essentially an async version of Enqueue that does not block the caller.
func (d *RequestDispatcher) EnqueueAsync(req RequestNoErr) {
	go func() {
		d.reqBucket.AcquireToken()
		req()
	}()
}

// Like EnqueueAsync, but sends any non-nil error returned by the executed req into errChan.
func (d *RequestDispatcher) EnqueueAsyncErr(req Request, errChan chan error) {
	go func() {
		if err := d.Enqueue(req); err != nil {
			errChan <- err
		}
	}()
}

// Attempts to execute req immediately if a token is available.
// Otherwise, it is queued asynchronously via EnqueueAsync.
func (d *RequestDispatcher) EnqueueOrAsync(req RequestNoErr) {
	if !d.reqBucket.TryAcquireToken() {
		d.EnqueueAsync(req) // no token available, queue until there is one
		return
	}

	req() // Got a token, just execute and block :)
}

// Burst executes reqs as fast as this dispatcher's bucket allows, draining all available tokens.
//
// If too little tokens were available to complete all requests with the initial burst,
// QueueAsync() is called for every remaining request to ensure they arrive, but won't block.
func (d *RequestDispatcher) Burst(reqs []RequestNoErr) {
	for _, req := range reqs {
		select {
		case <-d.reqBucket.tokens:
			req() // Run immediately if token available
		default:
			// Queue remaining requests asynchronously
			d.EnqueueAsync(req)
		}
	}
}

func (d *RequestDispatcher) GetBucketTokens() chan Token {
	return d.reqBucket.tokens
}
