package oapi

import (
	"emcsrw/utils"
	"sync"

	"github.com/samber/lo"
)

const PLAYERS_QUERY_LIMIT = 100

type NamesQuery struct {
	Query []string `json:"query"`
}

func NewNamesQuery(names ...string) NamesQuery {
	return NamesQuery{Query: names}
}

func QueryPlayers(identifiers ...string) ([]PlayerInfo, error) {
	return utils.OAPIPostRequest[[]PlayerInfo]("/players", NewNamesQuery(identifiers...))
}

// Queries players in chunks concurrently where each chunk (request) query may only have up to chunkSize identifiers.
// If there are leftover identifiers, they will be queried in a final request.
//
// The following example sends X amount of requests as necessary until each req has max 50 names in the query.
// If the chunkSize is 0, it defaults to the query limit for the players endpoint specified by PLAYERS_QUERY_LIMIT.
//
//	players, err := QueryPlayersConcurrent(names, 50)
//
// A [sync.WaitGroup] will catch the error that may occur during any of the requests and append it to the resulting error slice.
func QueryPlayersConcurrent(identifiers []string, chunkSize uint8) ([]PlayerInfo, []error) {
	if chunkSize == 0 {
		chunkSize = PLAYERS_QUERY_LIMIT
	}

	chunks := lo.Chunk(identifiers, int(chunkSize))
	chunkLen := len(chunks)

	all := make([]PlayerInfo, 0, len(identifiers))
	errCh := make(chan error, chunkLen)

	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(chunkLen)

	for _, chunk := range chunks {
		// Execute for each chunk. Queries OAPI and sends result to result channel.
		go func(c []string) {
			defer wg.Done()

			players, err := QueryPlayers(c...)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			all = append(all, players...)
			mu.Unlock()
		}(chunk)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	return all, errs
}

func QueryTowns(identifiers ...string) ([]TownInfo, error) {
	return utils.OAPIPostRequest[[]TownInfo]("/towns", NewNamesQuery(identifiers...))
}

func QueryNations(identifiers ...string) ([]NationInfo, error) {
	return utils.OAPIPostRequest[[]NationInfo]("/nations", NewNamesQuery(identifiers...))
}

func QueryServer() (RawServerInfoV3, error) {
	return utils.OAPIGetRequest[RawServerInfoV3]("")
}
