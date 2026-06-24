package capi

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha1"
	"emcsrw/internal/database"
	"emcsrw/internal/database/store"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/rs/cors"
)

var md = goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))

// Req/m for each endpoint
const (
	// FALLING_RPM   = 2
	// RUINED_RPM    = 4
	ALLIANCES_RPM = 8
	NEWS_RPM      = 6
	PLAYERS_RPM   = 3
	PROXY_RPM     = 30
)

const BASE_WELCOME_STR = `
Welcome to the Custom API! All info here is only available originally by the EarthMC Stats Discord bot.

To access data for a specific map, navigate to "https://emcstats.bot.nu/mapName/endpoint".
For example, "/nostra/alliances" for alliance data on the Nostra map.

The following endpoints are available:
- /falling
- /ruined
- /alliances
- /players
- /news
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

func ServeBase(mux *http.ServeMux) {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(strings.TrimPrefix(BASE_WELCOME_STR, "\n")))
	})
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		http.ServeFile(w, r, "./icon.png")
	})
}

func ServeTerms(mux *http.ServeMux) {
	mux.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile("TERMS.md")
		if err != nil {
			http.Error(w, "TERMS.md not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!doctype html><html><head><style>
			body{line-height:1.3rem;font-size:14px;font-family:monospace;background:#1a1c23f7;color:#ffffff;max-width:900px;margin:20px auto;padding:20px}
			strong{color:#cde2ff}
			h1{text-decoration:underline;color:#cde2ff}
			h2{margin-block-end:0px;color:#cde2ff}
			</style></head><body>`,
		))

		md.Convert(data, w) // Render the terms file (converts markdown to HTML).
		w.Write([]byte(`</body></html>`))
	})
}

func ServeBotInvite(mux *http.ServeMux) {
	mux.HandleFunc("/invite", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://discord.com/oauth2/authorize?client_id=656231016385478657", http.StatusFound)
	})
}

func ServeProxy(mux *http.ServeMux, proxy *Proxy) {
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedHeaders:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowCredentials: true,
		Debug:            false,
	})

	mux.Handle("/proxy", c.Handler(http.HandlerFunc(proxy.handler)))
}

func ServeFalling(
	mux *http.ServeMux, mdbName string,
	fallingTownStore *store.Store[database.FallingTown],
) {
	fallingEndpoint := fmt.Sprintf("/%s/falling", mdbName)
	mux.HandleFunc(fallingEndpoint, TTLGzipHandler(90*time.Second, func() []database.FallingTown {
		falling := fallingTownStore.Values()

		// Default sort (most inactive mayor first, then least amt of residents)
		utils.KeySort(falling, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool { return a.InactiveDuration > b.InactiveDuration }},
			{Compare: func(a, b database.FallingTown) bool { return a.NumResidents() < b.NumResidents() }},
		})

		return falling
	}))
}

func ServeRuined(
	mux *http.ServeMux, mdbName string,
	townStore *store.Store[oapi.TownInfo],
) {
	ruinedEndpoint := fmt.Sprintf("/%s/ruined", mdbName)
	mux.HandleFunc(ruinedEndpoint, TTLGzipHandler(90*time.Second, func() []oapi.TownInfo {
		return database.GetRuinedTowns(townStore)
	}))
}

func ServeAlliances(
	mux *http.ServeMux, rl *RateLimit, mdbName string,
	allianceStore *store.Store[database.Alliance],
	nationStore *store.Store[oapi.NationInfo],
	entitiesStore *store.Store[oapi.EntityList],
) {
	var alliancesCache = map[string]*CacheEntry{} // Cache response per map (key is map name)
	var alliancesCacheMu sync.RWMutex

	alliancesEndpoint := fmt.Sprintf("/%s/alliances", mdbName)
	mux.HandleFunc(alliancesEndpoint, func(w http.ResponseWriter, r *http.Request) {
		alliances := allianceStore.Values()

		// compute lightweight hash of alliance identifiers + timestamps
		currentHash := computeHash(alliances, func(a database.Alliance) (string, int64) {
			return a.Identifier, int64(*a.UpdatedTimestamp)
		})

		etag := `"` + currentHash + `"`
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
			limiter := rl.clientLimiter(r, ALLIANCES_RPM)
			if !limiter.Allow() {
				http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
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
}

func ServeNews(
	mux *http.ServeMux, rl *RateLimit, mdbName string,
	newsStore *store.Store[database.NewsEntry],
) {
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
		limiter := rl.clientLimiter(r, NEWS_RPM)
		if !limiter.Allow() {
			http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
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
}

func ServePlayers(
	mux *http.ServeMux, rl *RateLimit, mdbName string,
	playersStore *store.Store[database.BasicPlayer],
) {
	playersEndpoint := fmt.Sprintf("/%s/players", mdbName)
	mux.HandleFunc(playersEndpoint, func(w http.ResponseWriter, r *http.Request) {
		limiter := rl.clientLimiter(r, PLAYERS_RPM)
		if !limiter.Allow() {
			http.Error(w, "Rate Limit Exceeded", http.StatusTooManyRequests)
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
}

func TTLGzipHandler[T any](
	ttl time.Duration,
	dataFunc func() []T,
) http.HandlerFunc {
	var cache []byte
	var cacheExp time.Time
	var mu sync.RWMutex

	write := func(w http.ResponseWriter, data []byte) {
		w.Header().Set("Cache-Control", "public, max-age=90")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Write(data)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()

		mu.RLock()
		if now.Before(cacheExp) && cache != nil {
			data := cache
			mu.RUnlock()
			write(w, data)
			return
		}
		mu.RUnlock()

		mu.Lock()
		defer mu.Unlock()

		now = time.Now()
		if now.Before(cacheExp) && cache != nil {
			write(w, cache)
			return
		}

		dataSlice := dataFunc()
		data, err := gzipJSON(dataSlice, 2)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cache = data
		cacheExp = now.Add(ttl)

		write(w, data)
	}
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
