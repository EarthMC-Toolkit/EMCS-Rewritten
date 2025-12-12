package capi

import (
	"crypto/sha1"
	"emcsrw/database"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Req/m for "<mapName>/alliances" endpoint
const ALLIANCES_RPM = 3

func NewMux(mdb *database.Database) (*http.ServeMux, error) {
	allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
	if err != nil {
		return nil, err
	}

	nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
	if err != nil {
		return nil, err
	}

	entitiesStore, err := database.GetStore(mdb, database.ENTITIES_STORE)
	if err != nil {
		return nil, err
	}

	alliancesEndpoint := fmt.Sprintf("/%s/alliances", mdb.Name())

	mux := http.NewServeMux()
	mux.HandleFunc(alliancesEndpoint, func(w http.ResponseWriter, r *http.Request) {
		// Pre-cache common data
		reslist, _ := entitiesStore.GetKey("residentlist")
		townlesslist, _ := entitiesStore.GetKey("townlesslist")

		alliances := allianceStore.Values()
		parsedAlliances := getParsedAlliances(alliances, nationStore, reslist, townlesslist)

		// Generate SHA-1 ETag from JSON
		data, _ := json.Marshal(parsedAlliances)
		hash := sha1.Sum(data)
		etag := fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))

		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", "public, max-age=90")
		w.Header().Set("Content-Type", "application/json")

		if match := r.Header.Get("If-None-Match"); match != "" && match == etag {
			w.WriteHeader(http.StatusNotModified)
			return // cached response, don't touch limiter
		}

		limiter := getLimiter(r.RemoteAddr, ALLIANCES_RPM)
		if !limiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		w.Write(data)
	})

	return mux, nil
}

// Serves the API using a reverse proxy
func Serve(mux *http.ServeMux) *http.Server {
	// NOTE: If we want a clean URL without specifying port, we need use the default port.
	// These are 443 if using HTTPS (need cert) or 80 if using HTTP.
	s := &http.Server{
		Addr:    ":7777",
		Handler: mux,
	}

	go func() {
		log.Println("Custom API server listening on :7777")
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Custom API server error: %v", err)
		}
	}()

	return s
}
