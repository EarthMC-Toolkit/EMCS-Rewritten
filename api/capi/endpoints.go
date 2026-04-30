package capi

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha1"
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/samber/lo"
)

// Req/m for each endpoint
const (
	ALLIANCES_RPM = 8
	NEWS_RPM      = 6
	PLAYERS_RPM   = 3
)

const BASE_WELCOME_STR = `
Welcome to the Custom API! All info here is only available originally by the EarthMC Stats Discord bot.

To access data for a specific map, navigate to "https://emcstats.bot.nu/mapName/endpoint".
For example, "/nostra/alliances" for alliance data on the Nostra map.

The following endpoints are available:
- /alliances
- /news
- /players
`

type BasicPlayer struct {
	Name   string             `json:"name"`
	UUID   string             `json:"uuid"`
	Town   *[2]string         `json:"town,omitempty"`
	Nation *[2]string         `json:"nation,omitempty"`
	Rank   *database.RankType `json:"rank,omitempty"`
}

type NewsEntry struct {
	database.NewsEntry
	ID string `json:"id"`
}

// type ResponseCache struct {
// 	CompressedData []byte // gzip compressed data that gets sent (usually JSON)
// 	Hash           string // server-side internal fingerprint of the data
// 	ETag           string // like hash, but intended to match what the client still has
// }

type CacheEntry struct {
	Hash string
	Data []Alliance // parsed alliances
}

type DatabaseName = string

var alliancesCache = map[DatabaseName]*CacheEntry{} // Cache the response per map
var alliancesCacheMu sync.RWMutex

func ServeBase(mux *http.ServeMux) error {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.TrimPrefix(BASE_WELCOME_STR, "\n")))
	})

	return nil
}

func ServeTerms(mux *http.ServeMux) error {
	mux.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("TERMS.md")
		if err != nil {
			w.Write([]byte("Error serving TOS and Privacy Policy for EarthMC Stats. No TERMS.md file found."))
			return
		}

		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write(data)
	})

	return nil
}

func ServeAlliances(
	mux *http.ServeMux, mdbName string,
	allianceStore *store.Store[database.Alliance],
	nationStore *store.Store[oapi.NationInfo],
	entitiesStore *store.Store[oapi.EntityList],
) error {
	alliancesEndpoint := fmt.Sprintf("/%s/alliances", mdbName)
	mux.HandleFunc(alliancesEndpoint, func(w http.ResponseWriter, r *http.Request) {
		alliances := allianceStore.Values()

		// compute lightweight hash of alliance identifiers + timestamps
		currentHash := computeHash(alliances, func(a database.Alliance) (string, int64) {
			return a.Identifier, int64(*a.UpdatedTimestamp)
		})

		etag := fmt.Sprintf(`"%x"`, sha1.Sum([]byte(currentHash)))
		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		alliancesCacheMu.RLock()
		entry, ok := alliancesCache[mdbName]
		alliancesCacheMu.RUnlock()

		var parsedAlliances []Alliance
		if ok && entry.Hash == currentHash {
			parsedAlliances = entry.Data
		} else {
			// cache miss → apply limiter
			limiter := getLimiter(getClientIP(r), alliancesEndpoint, ALLIANCES_RPM)
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}

			reslist, _ := entitiesStore.Get("residentlist")
			townlesslist, _ := entitiesStore.Get("townlesslist")
			parsedAlliances = getParsedAlliances(alliances, nationStore, reslist, townlesslist)

			alliancesCacheMu.Lock()
			alliancesCache[mdbName] = &CacheEntry{
				Hash: currentHash,
				Data: parsedAlliances,
			}
			alliancesCacheMu.Unlock()
		}

		data, err := gzipJSON(parsedAlliances, 1)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Write(data)
	})

	return nil
}

func ServeNews(mux *http.ServeMux, mdbName string, newsStore *store.Store[database.NewsEntry]) error {
	newsEndpoint := fmt.Sprintf("/%s/news", mdbName)
	mux.HandleFunc(newsEndpoint, func(w http.ResponseWriter, r *http.Request) {
		newsValues := lo.MapToSlice(newsStore.Entries(), func(key string, n database.NewsEntry) NewsEntry {
			return NewsEntry{NewsEntry: n, ID: key}
		})
		slices.SortFunc(newsValues, func(a, b NewsEntry) int {
			return cmp.Compare(b.Timestamp, a.Timestamp) // newest first
		})

		b, _ := json.Marshal(newsValues)
		sum := sha1.Sum(b)
		etag := fmt.Sprintf(`"%x"`, sum)

		// 304 if client already has current snapshot
		if r.Header.Get("If-None-Match") == etag && etag != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// rate limit applies only when serving data
		limiter := getLimiter(getClientIP(r), newsEndpoint, NEWS_RPM)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		var buf bytes.Buffer
		gz, err := gzip.NewWriterLevel(&buf, 2)
		if err != nil {
			http.Error(w, "Gzip error: failed to create writer", http.StatusInternalServerError)
			return
		}
		if _, err := gz.Write(b); err != nil {
			http.Error(w, "Gzip error: failed to write", http.StatusInternalServerError)
			return
		}
		gz.Close()

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=120")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Write(buf.Bytes())
	})

	return nil
}

func ServePlayers(mux *http.ServeMux, mdbName string, playersStore *store.Store[database.BasicPlayer]) error {
	playersEndpoint := fmt.Sprintf("/%s/players", mdbName)
	mux.HandleFunc(playersEndpoint, func(w http.ResponseWriter, r *http.Request) {
		limiter := getLimiter(r.RemoteAddr, playersEndpoint, PLAYERS_RPM)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=30")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")

		gz, err := gzip.NewWriterLevel(w, 3)
		if err != nil {
			http.Error(w, "Error during gzip data compression", http.StatusInternalServerError)
			return
		}
		defer gz.Close()

		// Condense town and nation fields from entity object to array to reduce payload size
		playerStoreValues := lo.MapToSlice(playersStore.Entries(), func(key string, p database.BasicPlayer) BasicPlayer {
			var town, nation *[2]string
			if p.Town != nil {
				t := [2]string{p.Town.Name, p.Town.UUID}
				town = &t
			}
			if p.Nation != nil {
				n := [2]string{p.Nation.Name, p.Nation.UUID}
				nation = &n
			}

			return BasicPlayer{
				UUID:   key,
				Name:   p.Name,
				Rank:   p.Rank,
				Town:   town,
				Nation: nation,
			}
		})

		json.NewEncoder(gz).Encode(playerStoreValues)
	})

	return nil
}

func gzipJSON[T any](v T, level int) ([]byte, error) {
	buf := bytes.Buffer{}
	gz, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	if err := json.NewEncoder(gz).Encode(v); err != nil {
		gz.Close()
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func computeHash[T any](items []T, keyFn func(T) (id string, timestamp int64)) string {
	h := sha1.New()
	for _, item := range items {
		id, ts := keyFn(item)
		h.Write([]byte(id))
		var buf [8]byte
		binary.LittleEndian.PutUint64(buf[:], uint64(ts))
		h.Write(buf[:])
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}
