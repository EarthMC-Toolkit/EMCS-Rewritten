package capi

import (
	"context"
	"emcsrw/internal/database"
	"emcsrw/internal/database/store"
	"emcsrw/internal/shared"
	"emcsrw/pkg/api/oapi"
	"emcsrw/pkg/utils/config"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func Start() {
	activeMapDB := database.TryInit(shared.ACTIVE_MAP)
	auroraDB := database.TryInit(shared.SUPPORTED_MAPS.AURORA)

	port := config.GetApiPort()
	if IsRunning(port) {
		log.Fatalf("Custom API server already listening on :%d", port)
		return
	}

	mux, err := NewMux(activeMapDB, auroraDB)
	if err != nil {
		log.Fatalf("failed to start Custom API. failed to init mux.\n%s", err)
	}

	server := Serve(mux, config.GetApiPort())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-c

	logutil.Printf(logutil.YELLOW, "\n\nShutting down Custom API with signal: %s\n", strings.ToUpper(sig.String()))

	// Gracefully shutdown HTTP server that serves the Custom API.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logutil.Logf(logutil.RED, "ERR | could not gracefully shut down Custom API: %v", err)
	}
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
		logutil.Printf(logutil.BLUE, "Custom API server listening on :%d", port)
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logutil.Printf(logutil.RED, "Custom API server error: %v", err)
		}
	}()

	return s
}

func NewMux(mdbs ...*database.Database) (mux *http.ServeMux, err error) {
	apiRL := NewRateLimit(true, 2)    // per IP, per endpoint
	proxyRL := NewRateLimit(false, 3) // per IP
	proxy := NewProxy(proxyRL, PROXY_RPM, []string{"earthmc.net", "map.earthmc.net", "api.earthmc.net"})

	mux = http.NewServeMux()
	ServeBase(mux)         // Welcome endpoint at domain base (also shown for unknown endpoints).
	ServeTerms(mux)        // Legal jargon page including TOS and Privacy Policy. See TERMS.md file.
	ServeBotInvite(mux)    // A redirect to invite the bot through Discord.
	ServeProxy(mux, proxy) // Custom CORS proxy with auth. Client must specify X-Proxy-Key and SECRET_KEY must match.

	for _, mdb := range mdbs {
		if mdb == nil {
			logutil.Println(logutil.RED, "ERR | attempted to serve Custom API endpoints for a nil map database")
			continue
		}

		fallingTownStore, err := database.GetStore(mdb, database.FALLING_TOWNS_STORE)
		if err != nil {
			return nil, err
		}
		townStore, err := database.GetStore(mdb, database.TOWNS_STORE)
		if err != nil {
			return nil, err
		}
		nationStore, err := database.GetStore(mdb, database.NATIONS_STORE)
		if err != nil {
			return nil, err
		}
		allianceStore, err := database.GetStore(mdb, database.ALLIANCES_STORE)
		if err != nil {
			return nil, err
		}
		entitiesStore, err := database.GetStore(mdb, database.ENTITIES_STORE)
		if err != nil {
			return nil, err
		}
		playersStore, err := database.GetStore(mdb, database.PLAYERS_STORE)
		if err != nil {
			return nil, err
		}
		newsStore, err := database.GetStore(mdb, database.NEWS_STORE)
		if err != nil {
			return nil, err
		}

		dbName := mdb.Name()

		// Every interval, refresh the stores so they reflect their respective DB files (source of truth)
		StartStoreSync(
			30*time.Second, dbName,
			fallingTownStore, townStore, nationStore, allianceStore,
			entitiesStore, playersStore,
			newsStore,
		)

		ServeFalling(mux, dbName, fallingTownStore)
		ServeRuined(mux, dbName, townStore)
		ServeAlliances(mux, apiRL, dbName, allianceStore, nationStore, entitiesStore)
		ServePlayers(mux, apiRL, dbName, playersStore)
		ServeNews(mux, apiRL, dbName, newsStore)
	}

	return mux, nil
}

func StartStoreSync(
	interval time.Duration, mdbName string,
	fallingTownStore *store.Store[database.FallingTown],
	townStore *store.Store[oapi.TownInfo],
	nationStore *store.Store[oapi.NationInfo],
	allianceStore *store.Store[database.Alliance],
	entitiesStore *store.Store[oapi.EntityList],
	playersStore *store.Store[database.BasicPlayer],
	newsStore *store.Store[database.NewsEntry],
) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			logutil.Logln(logutil.BLUE, "Syncing stores with data from underlying DB files for map: ", mdbName)

			// The API has it's own version of the stores which don't match the bot, so loading from
			// the DB is required to keep data synced since we can't use stores from the bot process.
			_ = fallingTownStore.LoadFromFile()
			_ = townStore.LoadFromFile()
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
