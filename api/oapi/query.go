package oapi

import (
	"emcsrw/utils/requests"
	"sync"

	"github.com/samber/lo"
)

type Endpoint = string

const VERSION, MAP = "3", "aurora"
const (
	ENDPOINT_BASE           Endpoint = "https://api.earthmc.net/v" + VERSION + "/" + MAP
	ENDPOINT_MYSTERY_MASTER Endpoint = ENDPOINT_BASE + "/mm"
	ENDPOINT_TOWNS          Endpoint = ENDPOINT_BASE + "/towns"
	ENDPOINT_NATIONS        Endpoint = ENDPOINT_BASE + "/nations"
	ENDPOINT_PLAYERS        Endpoint = ENDPOINT_BASE + "/players"
	ENDPOINT_ONLINE         Endpoint = ENDPOINT_BASE + "/online"
	ENDPOINT_LOCATION       Endpoint = ENDPOINT_BASE + "/location"
	ENDPOINT_DISCORD        Endpoint = ENDPOINT_BASE + "/discord"
	ENDPOINT_QUARTERS       Endpoint = ENDPOINT_BASE + "/quarters"
	ENDPOINT_PLAYER_STATS   Endpoint = ENDPOINT_BASE + "/player-stats"
)

// Identifiable is a constraint for things with a UUID such as an Entity.
type Identifiable interface {
	GetUUID() string
}

// type QueryFunc[T any] func() (T, error)
type RequestResult[T any] struct {
	Result T
	Err    error
}

type GetQuery[T any] struct {
	endpoint Endpoint
}

func NewGetQuery[T any](endpoint Endpoint) *GetQuery[T] {
	return &GetQuery[T]{endpoint: endpoint}
}

func (q *GetQuery[T]) Execute() (T, error) {
	resCh := make(chan RequestResult[T], 1)
	Dispatcher.Enqueue(func() error {
		res, err := requests.JsonGet[T](q.endpoint)
		resCh <- RequestResult[T]{res, err}
		return err
	})

	r := <-resCh
	return r.Result, r.Err
}

type PostBody struct {
	Query    []string        `json:"query"`
	Template map[string]bool `json:"template,omitempty"`
}

func NewPostBody(ids []string, template map[string]bool) *PostBody {
	if template == nil {
		return &PostBody{Query: ids}
	}

	return &PostBody{Query: ids, Template: template}
}

type PostQuery[T any] struct {
	endpoint Endpoint
	body     *PostBody
}

func NewPostQuery[T any](endpoint string, body *PostBody) *PostQuery[T] {
	return &PostQuery[T]{
		endpoint: endpoint,
		body:     body,
	}
}

func (q *PostQuery[T]) WithTemplate(template map[string]bool) *PostQuery[T] {
	q.body.Template = template
	return q
}

func (q *PostQuery[T]) Execute() ([]T, error) {
	resCh := make(chan RequestResult[[]T], 1)
	Dispatcher.Enqueue(func() error {
		results, err := requests.JsonPost[[]T](q.endpoint, q.body)
		resCh <- RequestResult[[]T]{results, err}
		return err
	})

	res := <-resCh
	return res.Result, res.Err
}

func (q *PostQuery[T]) ExecuteConcurrent() ([]T, []error, int) {
	chunks := lo.Chunk(q.body.Query, QUERY_LIMIT)
	chunkLen := len(chunks)

	all := []T{}
	errCh := make(chan error, chunkLen)

	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(chunkLen)

	for _, chunk := range chunks {
		chunkCopy := chunk
		Dispatcher.EnqueueAsync(func() error {
			defer wg.Done()

			body := NewPostBody(chunkCopy, q.body.Template)
			results, err := requests.JsonPost[[]T](q.endpoint, body)
			if err != nil {
				errCh <- err
				return err
			}

			mu.Lock()
			all = append(all, results...)
			mu.Unlock()

			return nil
		})
	}

	wg.Wait()
	close(errCh)

	errs := make([]error, 0, len(errCh))
	for err := range errCh {
		errs = append(errs, err)
	}

	return all, errs, chunkLen
}

// Sends a ping, aka a HEAD request to url.
// An enum is returned indicating the type of error we encountered, in addition to an ok
// boolean which returns true on success (RESPONSE_STATUS_OK), anything else is false.
func Ping(url string) (requests.ResponseStatus, bool) {
	return requests.WithResponseStatus(requests.Head(url))
}

// Queries the Official API with a GET request to the given endpoint.
// According to docs, this should return a list of entities (name, uuid) relating to the type of said endpoint.
//
// Do not call this function if you expect something other than an [Entity] slice to be returned.
// For example, "QueryList(oapi.SERVER_ENDPOINT)" will fail with a type conversion error.
func QueryList(endpoint Endpoint) *GetQuery[[]Entity] {
	return NewGetQuery[[]Entity](endpoint)
}

// Queries the Official API with a GET request to the server endpoint.
func QueryServer() *GetQuery[ServerInfo] {
	return NewGetQuery[ServerInfo](ENDPOINT_BASE)
}

func QueryOnline() *GetQuery[OnlineResponse] {
	return NewGetQuery[OnlineResponse](ENDPOINT_ONLINE)
}

func QueryServerPlayerStats() *GetQuery[ServerPlayerStats] {
	return NewGetQuery[ServerPlayerStats](ENDPOINT_PLAYER_STATS)
}

func QueryMysteryMaster() *GetQuery[[]MysteryMaster] {
	return NewGetQuery[[]MysteryMaster](ENDPOINT_MYSTERY_MASTER)
}

// Queries the Official API with a POST request providing all valid town identifier (name/uuid) strings to the body "query" key.
func QueryTowns(identifiers ...string) *PostQuery[TownInfo] {
	return NewPostQuery[TownInfo](ENDPOINT_TOWNS, NewPostBody(identifiers, nil))
}

// Queries the Official API with a POST request providing all valid nation identifier (name/uuid) strings to the body "query" key.
func QueryNations(identifiers ...string) *PostQuery[NationInfo] {
	return NewPostQuery[NationInfo](ENDPOINT_NATIONS, NewPostBody(identifiers, nil))
}

// Queries the Official API with a POST request providing all valid player identifier (name/uuid) strings to the body "query" key.
func QueryPlayers(identifiers ...string) *PostQuery[PlayerInfo] {
	return NewPostQuery[PlayerInfo](ENDPOINT_PLAYERS, NewPostBody(identifiers, nil))
}

// Queries the Official API with a POST request providing all valid quarter identifier (name/uuid) strings to the body "query" key.
func QueryQuarters(identifiers ...string) *PostQuery[Quarter] {
	return NewPostQuery[Quarter](ENDPOINT_QUARTERS, NewPostBody(identifiers, nil))
}
