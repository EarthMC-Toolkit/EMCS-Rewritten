package capi

import (
	"emcsrw/database"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

type NewsEntry struct {
	database.NewsEntry
	ID string `json:"id"`
}

// Req/m for "<mapName>/alliances" endpoint
const ALLIANCES_RPM = 5
const PLAYERS_RPM = 3

func NewMux(mdb *database.Database) (mux *http.ServeMux, err error) {
	mux = http.NewServeMux()

	if err = ServeBase(mux); err != nil {
		return nil, err
	}
	if err = ServeAlliances(mux, mdb); err != nil {
		return nil, err
	}
	if err = ServePlayers(mux, mdb); err != nil {
		return nil, err
	}
	if err = ServeNews(mux, mdb); err != nil {
		return nil, err
	}

	return mux, nil
}

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
		log.Println("Custom API server listening on :" + fmt.Sprint(port))
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Custom API server error: %v", err)
		}
	}()

	return s
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
