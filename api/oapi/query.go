package oapi

import (
	"emcsrw/utils"
	"sync"
	"time"

	"github.com/samber/lo"
)

const RATE_LIMIT = 180          // 180 req/min
const PLAYERS_QUERY_LIMIT = 100 // 100 identifiers in single req/query

type NamesQuery struct {
	Query []string `json:"query"`
}

func NewNamesQuery(names ...string) NamesQuery {
	return NamesQuery{Query: names}
}

func QueryPlayersList() ([]Entity, error) {
	return utils.OAPIGetRequest[[]Entity]("/players")
}

func QueryPlayers(identifiers ...string) ([]PlayerInfo, error) {
	return utils.OAPIPostRequest[[]PlayerInfo]("/players", NewNamesQuery(identifiers...))
}

// Queries players in chunks concurrently where each chunk (request) query may only have up to PLAYERS_QUERY_LIMIT identifiers.
// If there are leftover identifiers, they will be queried in a final request.
//
// If you only need to send one request, use QueryPlayers instead.
//
// The following example sends X amount of requests as necessary and sleeps for 200ms between each request.
//
//	players, err := QueryPlayersConcurrent(names, 200)
//
// A [sync.WaitGroup] will catch the error that may occur during any of the requests and append it to the resulting error slice.
func QueryPlayersConcurrent(identifiers []string, sleepAmt uint32) ([]PlayerInfo, []error, int) {
	chunks := lo.Chunk(identifiers, int(PLAYERS_QUERY_LIMIT))
	chunkLen := len(chunks)

	all := make([]PlayerInfo, 0, len(identifiers))
	errCh := make(chan error, chunkLen)

	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	wg.Add(chunkLen)

	var ticker *time.Ticker
	if sleepAmt > 0 {
		ticker = time.NewTicker(time.Duration(sleepAmt * uint32(time.Millisecond))) // Ticker to avoid rate limit.
		defer ticker.Stop()
	}

	for i, chunk := range chunks {
		if ticker != nil && i > 0 { // Don't add delay before first req.
			<-ticker.C
		}

		// Execute for each chunk. Queries OAPI and sends result to result channel.
		go func(arr []string) {
			defer wg.Done()

			players, err := QueryPlayers(arr...)
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

	// Deduplicate by UUID, keeping the last occurrence
	playerMap := make(map[string]PlayerInfo)
	for _, p := range all {
		playerMap[p.UUID] = p
	}

	unique := make([]PlayerInfo, 0, len(playerMap))
	for _, p := range playerMap {
		unique = append(unique, p)
	}

	return unique, errs, chunkLen
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
