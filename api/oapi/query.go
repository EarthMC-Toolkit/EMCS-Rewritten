package oapi

import (
	"emcsrw/utils/requests"
	"strings"
	"sync"

	"github.com/samber/lo"
	lop "github.com/samber/lo/parallel"
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

// Sends a ping, aka a HEAD request to url.
// An enum is returned indicating the type of error we encountered, in addition to an ok
// boolean which returns true on success (RESPONSE_STATUS_OK), anything else is false.
func Ping(url string) (requests.ResponseStatus, bool) {
	return requests.WithResponseStatus(requests.Head(url))
}

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
	return requests.JsonGet[[]Entity](endpoint)
}

// Queries the Official API with a GET request to the server endpoint.
func QueryServer() (ServerInfo, error) {
	return requests.JsonGet[ServerInfo](ENDPOINT_BASE)
}

func QueryServerPlayerStats() (ServerPlayerStats, error) {
	return requests.JsonGet[ServerPlayerStats](ENDPOINT_PLAYER_STATS)
}

func QueryMysteryMaster() ([]MysteryMaster, error) {
	return requests.JsonGet[[]MysteryMaster](ENDPOINT_MYSTERY_MASTER)
}

// Queries the Official API with a POST request providing all valid town identifier (name/uuid) strings to the body "query" key.
func QueryTowns(identifiers ...string) ([]TownInfo, error) {
	return requests.JsonPost[[]TownInfo](ENDPOINT_TOWNS, NewPostQuery(identifiers...))
}

// Queries the Official API with a POST request providing all valid nation identifier (name/uuid) strings to the body "query" key.
func QueryNations(identifiers ...string) ([]NationInfo, error) {
	return requests.JsonPost[[]NationInfo](ENDPOINT_NATIONS, NewPostQuery(identifiers...))
}

// Queries the Official API with a POST request providing all valid player identifier (name/uuid) strings to the body "query" key.
func QueryPlayers(identifiers ...string) ([]PlayerInfo, error) {
	return requests.JsonPost[[]PlayerInfo](ENDPOINT_PLAYERS, NewPostQuery(identifiers...))
}

// Queries the Official API with a POST request providing all valid quarter identifier (name/uuid) strings to the body "query" key.
func QueryQuarters(identifiers ...string) ([]Quarter, error) {
	return requests.JsonPost[[]Quarter](ENDPOINT_QUARTERS, NewPostQuery(identifiers...))
}

func QuerySequential[T Identifiable](
	queryFunc func(ids ...string) ([]T, error),
	identifiers []string,
	stopOnError bool,
) ([]T, []error) {
	all := make([]T, 0, len(identifiers))
	errs := []error{}

	chunks := lo.Chunk(identifiers, QUERY_LIMIT)
	for _, chunk := range chunks {
		results, err := queryFunc(chunk...)
		if err != nil {
			errs = append(errs, err)
			if stopOnError {
				return all, errs // stop immediately if requested
			}

			continue // skip this chunk. results not needed bc we errored
		}

		all = append(all, results...)
	}

	return all, errs
}

// Queries entities of type T in chunks concurrently where each chunk (request) query may only have up to QUERY_LIMIT identifiers.
// If there are any leftover identifiers, they will be queried in a final request.
// A [sync.WaitGroup] will catch the error that may occur during any of the requests and append it to the resulting error slice.
//
// This func has slight overhead and calling queryFunc directly where possible is always preferred!
func QueryConcurrent[T Identifiable](
	queryFunc func(ids ...string) ([]T, error),
	identifiers []string,
) ([]T, []error, int) {
	chunks := lo.Chunk(identifiers, QUERY_LIMIT)
	chunkLen := len(chunks)

	all := make([]T, 0, len(identifiers))
	errCh := make(chan error, chunkLen)

	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	wg.Add(chunkLen)

	for _, chunk := range chunks {
		chunkCopy := chunk // capture so goroutine can access it
		Dispatcher.EnqueueAsyncErr(func() error {
			defer wg.Done()

			results, err := queryFunc(chunkCopy...)
			if err != nil {
				return err
			}

			mu.Lock()
			all = append(all, results...)
			mu.Unlock()

			return nil
		}, errCh)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return all, errs, chunkLen
}

func QueryConcurrentEntities[T Identifiable](
	queryFunc func(ids ...string) ([]T, error),
	entities []Entity,
) ([]T, []error, int) {
	ids := lop.Map(entities, func(e Entity, _ int) string {
		return e.UUID
	})

	return QueryConcurrent(queryFunc, ids)
}

func QueryTown(townName string) (*TownInfo, error) {
	towns, err := QueryTowns(strings.ToLower(townName))
	if err != nil {
		return nil, err
	}

	if len(towns) == 0 {
		return nil, nil
	}

	t := towns[0]
	return &t, nil
}

func QueryNation(nationName string) (*NationInfo, error) {
	nations, err := QueryNations(strings.ToLower(nationName))
	if err != nil {
		return nil, err
	}

	if len(nations) == 0 {
		return nil, nil
	}

	n := nations[0]
	return &n, nil
}
