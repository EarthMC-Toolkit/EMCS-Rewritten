package capi

import (
	"emcsrw/api/oapi"
	"emcsrw/database"
	"emcsrw/database/store"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Serves the API using on localhost at port.
//
// See README.md for details on how to setup a reverse proxy to a domain.
func Serve(mux *http.ServeMux, port uint) *http.Server {
	// NOTE: If we want a clean URL without specifying port, we need use the default port.
	// These are 443 if using HTTPS (need cert) or 80 if using HTTP.
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		log.Printf("Custom API server listening on :%d", port)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Custom API server error: %v", err)
		}
	}()

	return s
}

func NewMux(mdbs []*database.Database) (mux *http.ServeMux, err error) {
	mux = http.NewServeMux()
	if err = ServeBase(mux); err != nil {
		return nil, err
	}
	if err = ServeTerms(mux); err != nil {
		return nil, err
	}

	for _, mdb := range mdbs {
		if mdb == nil {
			log.Print("ERR | attempted to serve Custom API endpoints for a nil map database")
			continue
		}

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
		newsStore, err := database.GetStore(mdb, database.NEWS_STORE)
		if err != nil {
			return nil, err
		}
		playersStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
		if err != nil {
			return nil, err
		}

		dbName := mdb.Name()

		// Every interval, refresh the stores so they reflect their respective DB files (source of truth)
		StartStoreSync(
			30*time.Second, dbName,
			allianceStore, nationStore, entitiesStore,
			newsStore, playersStore,
		)

		if err = ServeAlliances(mux, dbName, allianceStore, nationStore, entitiesStore); err != nil {
			return nil, err
		}
		if err = ServeNews(mux, dbName, newsStore); err != nil {
			return nil, err
		}
		if err = ServePlayers(mux, dbName, playersStore); err != nil {
			return nil, err
		}
	}

	return mux, nil
}

func StartStoreSync(
	interval time.Duration, mdbName string,
	allianceStore *store.Store[database.Alliance],
	nationStore *store.Store[oapi.NationInfo],
	entitiesStore *store.Store[oapi.EntityList],
	newsStore *store.Store[database.NewsEntry],
	playersStore *store.Store[database.BasicPlayer],
) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Syncing stores with data from underlying DB files for map: ", mdbName)

			_ = allianceStore.LoadFromFile()
			_ = nationStore.LoadFromFile()
			_ = entitiesStore.LoadFromFile()
			_ = newsStore.LoadFromFile()
			_ = playersStore.LoadFromFile()
		}
	}()
}

func IsRunning(port uint) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false // cannot connect → API not running
	}

	conn.Close()
	return true
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs, we just need the first
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Fallback to RemoteAddr and strip port
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // unlikely, but just in case
	}

	return host
}
