package capi

import (
	"cmp"
	"crypto/sha1"
	"emcsrw/database"
	"emcsrw/utils"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
)

const BASE_WELCOME_STR = `
Welcome to the Custom API! All info here is only available originally by the EarthMC Stats Discord bot.

To access data for a specific map, navigate to "https://emcstats.bot.nu/mapName/endpoint".
For example, "/aurora/alliances" for alliance data on the Aurora map.

The following endpoints are available:
- /alliances
- /players
- /news
`

func ServeBase(mux *http.ServeMux) error {
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(BASE_WELCOME_STR))
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
		// Pre-cache common data
		reslist, _ := entitiesStore.Get("residentlist")
		townlesslist, _ := entitiesStore.Get("townlesslist")

		alliances := allianceStore.Values()
		//rankedAlliances := database.GetRankedAlliances(allianceStore, nationStore, database.DEFAULT_ALLIANCE_WEIGHTS)
		parsedAlliances := getParsedAlliances(alliances, nationStore, reslist, townlesslist)

		// Generate ETag from top level alliance data
		// to avoid sending data if nothing changed
		h := sha1.New()
		for _, a := range parsedAlliances {
			h.Write([]byte(a.Identifier))
			binary.Write(h, binary.LittleEndian, a.UpdatedTimestamp)
		}
		etag := fmt.Sprintf(`"%x"`, h.Sum(nil))

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Content-Type", "application/json")

		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return // cached response, don't touch limiter
		}

		limiter := getLimiter(getClientIP(r), alliancesEndpoint, ALLIANCES_RPM)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		data, _ := json.Marshal(parsedAlliances)
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
		w.Header().Set("Cache-Control", "public, max-age=30")
		w.Header().Set("Content-Type", "application/json")

		limiter := getLimiter(r.RemoteAddr, playersEndpoint, PLAYERS_RPM)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		data, _ := json.Marshal(playerStore.Values())
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
		w.Header().Set("Cache-Control", "public, max-age=120")
		w.Header().Set("Content-Type", "application/json")

		newsValues := utils.MapValues(newsStore.Entries(), func(id string, n database.NewsEntry) NewsEntry {
			return NewsEntry{NewsEntry: n, ID: id}
		})
		slices.SortFunc(newsValues, func(a, b NewsEntry) int {
			return cmp.Compare(b.Timestamp, a.Timestamp) // sort news by acsending (newest first)
		})

		data, _ := json.Marshal(newsValues)
		w.Write(data)
	})

	return nil
}
