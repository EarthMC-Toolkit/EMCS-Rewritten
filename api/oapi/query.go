package oapi

import (
	"emcsrw/utils"
	"sync"

	"github.com/samber/lo"
)

type Endpoint = string

const (
	ENDPOINT_BASE           Endpoint = "https://api.earthmc.net/v3/aurora"
	ENDPOINT_MYSTERY_MASTER Endpoint = ENDPOINT_BASE + "/mm"
	ENDPOINT_TOWNS          Endpoint = ENDPOINT_BASE + "/towns"
	ENDPOINT_NATIONS        Endpoint = ENDPOINT_BASE + "/nations"
	ENDPOINT_PLAYERS        Endpoint = ENDPOINT_BASE + "/players"
	ENDPOINT_LOCATION       Endpoint = ENDPOINT_BASE + "/location"
	ENDPOINT_DISCORD        Endpoint = ENDPOINT_BASE + "/discord"
	ENDPOINT_QUARTERS       Endpoint = ENDPOINT_BASE + "/quarters"
)

// Identifiable is a constraint for things with a UUID such as an Entity.
type Identifiable interface {
	GetUUID() string
}

type PostBody struct {
	Query []string `json:"query"`
}

type PostBodyTemplate[T any] struct {
	*PostBody
	Template T
}

func NewPostQuery(identifiers ...string) *PostBody {
	return &PostBody{Query: identifiers}
}

func NewPostQueryTemplate[T any](identifiers []string, template T) *PostBodyTemplate[T] {
	return &PostBodyTemplate[T]{
		Template: template,
		PostBody: &PostBody{
			Query: identifiers,
		},
	}
}

// Queries the Official API with a GET request to the given endpoint.
// According to docs, this should return a list of entities (name, uuid) relating to the type of said endpoint.
//
// Do not call this function if you do not expect an [Entity] slice back. For example QueryList(oapi.SERVER_ENDPOINT) will fail.
func QueryList(endpoint Endpoint) ([]Entity, error) {
	return utils.JsonGetRequest[[]Entity](endpoint)
}

// Queries the Official API with a GET request to the server endpoint.
func QueryServer() (ServerInfo, error) {
	return utils.JsonGetRequest[ServerInfo](ENDPOINT_BASE)
}

func QueryMysteryMaster() ([]MysteryMaster, error) {
	return utils.JsonGetRequest[[]MysteryMaster](ENDPOINT_MYSTERY_MASTER)
}

// Queries the Official API with a POST request providing all valid town identifier (name/uuid) strings to the body "query" key.
func QueryTowns(identifiers ...string) ([]TownInfo, error) {
	return utils.JsonPostRequest[[]TownInfo](ENDPOINT_TOWNS, NewPostQuery(identifiers...))
}

// Queries the Official API with a POST request providing all valid nation identifier (name/uuid) strings to the body "query" key.
func QueryNations(identifiers ...string) ([]NationInfo, error) {
	return utils.JsonPostRequest[[]NationInfo](ENDPOINT_NATIONS, NewPostQuery(identifiers...))
}

// Queries the Official API with a POST request providing all valid player identifier (name/uuid) strings to the body "query" key.
func QueryPlayers(identifiers ...string) ([]PlayerInfo, error) {
	return utils.JsonPostRequest[[]PlayerInfo](ENDPOINT_PLAYERS, NewPostQuery(identifiers...))
}

// Queries entities of type T in chunks concurrently where each chunk (request) query may only have up to QUERY_LIMIT identifiers.
// If there are any leftover identifiers, they will be queried in a final request.
// A [sync.WaitGroup] will catch the error that may occur during any of the requests and append it to the resulting error slice.
//
// The following example sends X amount of requests as necessary and sleeps for a 350ms (custom) between each request. To send requests
// with the default dynamic amount of sleep depending on the amount of items to keep under RATE_LIMIT, pass 0. For no sleep at all, pass any negative value.
//
//	players, err := oapi.QueryConcurrent(names, 350, oapi.QueryPlayers)
//
// This func has slight overhead and calling queryFunc directly where possible is always preferred!
func QueryConcurrent[T Identifiable](
	identifiers []string,
	queryFunc func(ids ...string) ([]T, error),
) ([]T, []error, int) {
	chunks := lo.Chunk(identifiers, QUERY_LIMIT)
	chunkLen := len(chunks)

	all := make([]T, 0, len(identifiers))
	errCh := make(chan error, chunkLen)

	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(chunkLen)

	for _, chunk := range chunks {
		chunkCopy := chunk // So that queue knows which chunk to work on.
		Dispatcher.Queue(func() {
			defer wg.Done()

			results, err := queryFunc(chunkCopy...)
			if err != nil {
				errCh <- err
			}

			mu.Lock()
			all = append(all, results...)
			mu.Unlock()
		})
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return all, errs, chunkLen
}
