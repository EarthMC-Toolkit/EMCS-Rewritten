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

const TOS_STR = `# Terms of Service for EarthMC Stats

**Effective Date:** April 30, 2026

## 1. Acceptance of Terms
By accessing or using EarthMC Stats, including its Discord bot, website, and API (the "Service"), you agree to these Terms of Service. If you do not agree, you must not use the Service.

## 2. Description of Service
EarthMC Stats provides statistics, analytics, and informational tools relating to the EarthMC Minecraft server. The Service may include data about players, towns, nations, alliances, maps, and news.
EarthMC Stats is an independent project and is not affiliated with, endorsed by, or sponsored by EarthMC, Minecraft, Mojang Studios, Microsoft, or Discord.

## 3. Permitted Use
You may use the Service for lawful, personal, and non-commercial purposes. You may not:

- Use the Service for unlawful purposes.
- Abuse, excessively scrape, disrupt, or interfere with the Service.

## 4. Intellectual Property
The software, branding, design, and original content of EarthMC Stats are protected by applicable intellectual property laws. You may not copy, redistribute, or create derivative works from the Service without prior permission.
Public EarthMC data remains the property of its respective owners.

## 5. Availability
The Service is provided on an "as is" and "as available" basis. Availability, features, and functionality may change at any time without notice.

## 6. Limitation of Liability
To the fullest extent permitted by law, EarthMC Stats shall not be liable for any direct, indirect, incidental, special, or consequential damages arising from your use of the Service.

## 7. Termination
Access to the Service may be suspended or terminated at any time for any reason, including abuse or violation of these Terms.

## 8. Changes to These Terms
These Terms may be updated periodically. Continued use of the Service after changes take effect constitutes acceptance of the revised Terms.

## 9. Contact
For questions regarding these Terms, contact Owen (EarthMC Stats developer) via Discord at @owen3h.
`

const PRIVACY_STR = `# Privacy Policy for EarthMC Stats

**Effective Date:** April 30, 2026

## 1. Information Collected
EarthMC Stats stores only limited information necessary to operate the Service, including:

- Publicly available EarthMC and Minecraft data, such as usernames, player statistics, towns, nations, alliances, and related game information.
- Publicly accessible messages and attachments from designated EarthMC news channels.
- Technical data required for API operation, such as temporary rate-limiting and caching information.

## 2. How Information Is Used
Collected information is used solely to provide, maintain, and improve EarthMC Stats and its features.

## 3. Data Sharing
EarthMC Stats does not sell, rent, trade, or share personal data with third parties.

## 4. Data Storage
Only publicly available game-related information and public news content are permanently stored. EarthMC Stats does not intentionally collect private personal information.

## 5. Third-Party Services
The Service may rely on third-party platforms, including Discord and EarthMC. Your use of those services is governed by their respective policies.

## 6. Data Removal
Given Sections 3 and 4, individual data deletion requests are generally not applicable. Requests to change a Minecraft username or public account information should be directed to Mojang Studios or Microsoft, and any such changes will be reflected automatically by the Service. Where reasonably possible, historical username records may be removed upon request.

## 7. Changes to This Policy
This Privacy Policy may be updated from time to time. Continued use of the Service constitutes acceptance of any revised policy.

## 8. Contact
For privacy-related questions, contact Owen (EarthMC Stats developer) via Discord at @owen3h.
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
	tos := strings.TrimPrefix(TOS_STR, "\n")
	privacy := strings.TrimPrefix(PRIVACY_STR, "\n")
	mux.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Write([]byte(tos + "\n\n" + privacy))
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
