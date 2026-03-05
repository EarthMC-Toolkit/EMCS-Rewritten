package oapi

import (
	"time"
)

const RATE_LIMIT = 180  // Amt req/min
const QUERY_LIMIT = 100 // Amt identifiers in single req/query

// The global dispatcher for queueing requests while adhering to RATE_LIMIT.
// Stores its own internal token bucket that is automatically refilled at RATE_LIMIT / 60.
//
// Any pending requests must wait for a token to be acquired before executing.
// Initializing the dispatcher with a pre-filled high capacity bucket may cause issues!
var Dispatcher *RequestDispatcher

func init() {
	Dispatcher = NewRequestDispatcher(RATE_LIMIT)
}

type Request func() error
type RequestNoErr func()
type Token struct{}

// A bucket of "tokens" where each token can allow a request to be sent.
// The bucket is responsible for refilling its tokens until the max amount (RATE_LIMIT).
type RequestBucket struct {
	tokens chan Token
}

// Allocates a new [RequestBucket] and initializes it by prefilling the bucket with tokens.
// It then starts a ticker which slowly refills the bucket with tokens at the refill rate specified by reqPerMin (converted to req/s).
func NewRequestBucket(reqPerMin int) *RequestBucket {
	reqPerMin = min(reqPerMin, RATE_LIMIT)
	reqPerSec := reqPerMin / 60 // Eg: 180req/m becomes 3req/s

	bucket := &RequestBucket{
		tokens: make(chan Token, reqPerSec),
	}
	for range reqPerSec {
		bucket.tokens <- Token{}
	}

	// Token refill rate. Eg: 3req/s becomes 333ms between ticks.
	refillRate := (time.Second / time.Duration(reqPerSec))
	ticker := time.NewTicker(refillRate + (time.Millisecond * 5)) // Add a 5ms safety margin.
	go func() {
		for range ticker.C {
			select {
			case bucket.tokens <- Token{}:
			default:
			}
		}
	}()

	return bucket
}

// Blocks the goroutine that this func was called in until a token/request is available and consumes it.
func (b *RequestBucket) WaitForToken() {
	// if len(b.tokens) == 0 {
	// 	fmt.Println("DEBUG | Request bucket empty — waiting for token")
	// }

	<-b.tokens
}

// Attempts to acquire a token without blocking the goroutine, reporting whether acquisition succeeded.
func (b *RequestBucket) TryWaitForToken() bool {
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

func NewRequestDispatcher(rateLimit int) *RequestDispatcher {
	return &RequestDispatcher{reqBucket: NewRequestBucket(rateLimit)}
}

func (d *RequestDispatcher) GetBucketTokens() chan Token {
	return d.reqBucket.tokens
}

// Executes the req synchronously after waiting to acquire a token, both of which block the caller.
//
// To make an async request, prefer EnqueueAsync or EnqueueAsyncErr for error logging.
func (d *RequestDispatcher) Enqueue(req Request) error {
	d.reqBucket.WaitForToken()
	return req()
}

// Runs a goroutine that handles executing req once a token is acquired.
// This is essentially an async version of Enqueue that does not block the caller.
func (d *RequestDispatcher) EnqueueAsync(req Request) {
	go func() {
		d.Enqueue(req)
	}()
}
