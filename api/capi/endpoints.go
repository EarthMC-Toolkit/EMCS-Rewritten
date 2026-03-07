package capi

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha1"
	"emcsrw/database"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
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
For example, "/aurora/alliances" for alliance data on the Aurora map.

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

type ResponseCache struct {
	CompressedData []byte // gzip compressed data that gets sent (usually JSON)
	Hash           string // server-side internal fingerprint of the data
	ETag           string // like hash, but intended to match what the client still has
}

var alliancesCache = &ResponseCache{}
var alliancesCacheMu = sync.RWMutex{}

var newsCache = &ResponseCache{}
var newsCacheMu = sync.RWMutex{}

func ServeBase(mux *http.ServeMux) error {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.TrimPrefix(BASE_WELCOME_STR, "\n")))
	})

	return nil
}

func ServeAlliances(mux *http.ServeMux, mdb *database.Database) error {
	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return err
	}

	entitiesStore, err := database.GetStore(mdb, database.ENTITIES_STORE)
	if err != nil {
		return err
	}

	alliancesEndpoint := fmt.Sprintf("/%s/alliances", mdb.Name())
	mux.HandleFunc(alliancesEndpoint, func(w http.ResponseWriter, r *http.Request) {
		alliances := allianceStore.Values()

		// compute lightweight hash of alliance identifiers + timestamps
		currentHash := computeHash(alliances, func(a database.Alliance) (string, int64) {
			return a.Identifier, int64(*a.UpdatedTimestamp)
		})

		alliancesCacheMu.RLock()
		cacheHit := alliancesCache.Hash == currentHash
		alliancesCacheMu.RUnlock()

		// rebuild cache only if hash changed
		if !cacheHit {
			reslist, _ := entitiesStore.Get("residentlist")
			townlesslist, _ := entitiesStore.Get("townlesslist")
			parsed := getParsedAlliances(alliances, nationStore, reslist, townlesslist)

			data, err := gzipJSON(parsed, 1)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			alliancesCacheMu.Lock()
			alliancesCache.CompressedData = data
			alliancesCache.ETag = fmt.Sprintf(`"%x"`, sha1.Sum(data))
			alliancesCache.Hash = currentHash
			alliancesCacheMu.Unlock()
		}

		// read cache after potential rebuild
		alliancesCacheMu.RLock()
		etag := alliancesCache.ETag
		data := alliancesCache.CompressedData
		alliancesCacheMu.RUnlock()

		// serve 304 if ETag matches
		if r.Header.Get("If-None-Match") == etag && etag != "" {
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// limiter applies only if uncached
		if !cacheHit {
			limiter := getLimiter(getClientIP(r), alliancesEndpoint, ALLIANCES_RPM)
			if !limiter.Allow() {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
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

func ServeNews(mux *http.ServeMux, mdb *database.Database) error {
	newsStore, err := database.GetStore(mdb, database.NEWS_STORE)
	if err != nil {
		return err
	}

	newsEndpoint := fmt.Sprintf("/%s/news", mdb.Name())
	mux.HandleFunc(newsEndpoint, func(w http.ResponseWriter, r *http.Request) {
		newsValues := lo.MapToSlice(newsStore.Entries(), func(key string, n database.NewsEntry) NewsEntry {
			return NewsEntry{NewsEntry: n, ID: key}
		})
		slices.SortFunc(newsValues, func(a, b NewsEntry) int {
			return cmp.Compare(b.Timestamp, a.Timestamp) // newest first
		})

		// gzip JSON to a buffer first
		var buf bytes.Buffer
		gz, err := gzip.NewWriterLevel(&buf, 2)
		if err != nil {
			http.Error(w, "Error during gzip compression", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(gz).Encode(newsValues); err != nil {
			gz.Close()
			http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
			return
		}
		gz.Close()
		data := buf.Bytes()

		// generate ETag from gzipped JSON
		etag := fmt.Sprintf(`"%x"`, sha1.Sum(data))

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

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=120")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Write(data)
	})

	return nil
}

func ServePlayers(mux *http.ServeMux, mdb *database.Database) error {
	playerStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
	if err != nil {
		return err
	}

	playersEndpoint := fmt.Sprintf("/%s/players", mdb.Name())
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
		playerStoreValues := lo.MapToSlice(playerStore.Entries(), func(key string, p database.BasicPlayer) BasicPlayer {
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
