package events

import (
	"emcsrw/api"
	"emcsrw/api/oapi"
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/slashcommands"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dgraph-io/badger/v4"
	"github.com/samber/lo"
)

// TODO: We shouldn't really be registering commands here bc of the limit.
// Prefer a standalone script that registers via the REST API and run it when a command "definition" changes (such as description, options etc).
// To see changes after modifying the `Execute` func, just restart the client without running said script.
//
// https://discordjs.guide/creating-your-bot/command-deployment.html#command-registration.
func OnReady(s *discordgo.Session, r *discordgo.Ready) {
	fmt.Printf("Logged in as: %s\n\n", s.State.User.Username)
	slashcommands.SyncWithRemote(s)

	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	if db == nil {
		fmt.Printf("[OnReady]: wtf happened? db is nil")
		return
	}

	// scheduleTask(func() {
	// 	PutFunc(db, "playerlist", func() ([]oapi.Entity, error) {
	// 		return oapi.QueryList(oapi.ENDPOINT_PLAYERS)
	// 	})
	// }, true, 20*time.Second)

	scheduleTask(func() {
		start := time.Now()

		towns, err := PutFunc(db, "towns", func() ([]oapi.TownInfo, error) {
			return api.QueryAllTowns()
		})
		if err != nil {
			return
		}

		//region ============ GATHER DATA USING TOWNS ============
		residentlist := make(map[string]oapi.Entity)
		nationlist := make(map[string]oapi.Entity)
		for _, t := range towns {
			for _, r := range t.Residents {
				residentlist[r.UUID] = r
			}

			if t.Nation.UUID != nil {
				nationlist[*t.Nation.UUID] = oapi.Entity{
					Name: *t.Nation.Name,
					UUID: *t.Nation.UUID,
				}
			}
		}

		PutFunc(db, "residentlist", func() (entities map[string]oapi.Entity, err error) {
			return residentlist, nil
		})

		nations, _ := PutFunc(db, "nations", func() ([]oapi.NationInfo, error) {
			res, _, _ := oapi.QueryConcurrent(lo.Keys(nationlist), oapi.QueryNations)
			return res, nil
		})
		//endregion

		//region ============ SPLIT RESIDENTS FROM TOWNLESS ============
		plist, err := oapi.QueryList(oapi.ENDPOINT_PLAYERS)
		if err != nil {
			return
		}

		townlesslist, _ := PutFunc(db, "townlesslist", func() (map[string]oapi.Entity, error) {
			entities := lo.FilterMap(plist, func(p oapi.Entity, _ int) (oapi.Entity, bool) {
				_, ok := residentlist[p.UUID]
				return p, !ok
			})

			// Convert slice to map using UUID as key.
			return lo.Associate(entities, func(p oapi.Entity) (string, oapi.Entity) {
				return p.UUID, p
			}), nil
		})
		//endregion

		// joined, left := EntityMapsDifference(townless, *staleTownless)
		// fmt.Printf("\nJoined a town: %s\n", strings.Join(joined, ", "))
		// fmt.Printf("\nLeft a town: %s\n", strings.Join(left, ", "))

		fmt.Printf("\nTotal Players: %d, Residents: %d, Townless: %d", len(plist), len(residentlist), len(townlesslist))
		fmt.Printf("\nNations List: %d, Nations: %d", len(nationlist), len(nations))
		fmt.Printf("\n\nTook: %s\n\n", time.Since(start))
	}, true, 40*time.Second)

	// Updating every 1m30s should be fine. doubt people are running /vp or /serverinfo that often.
	scheduleTask(func() {
		PutFunc(db, "serverinfo", func() (oapi.ServerInfo, error) {
			return oapi.QueryServer()
		})
	}, true, 90*time.Second)
}

func scheduleTask(task func(), runInitial bool, interval time.Duration) chan struct{} {
	if runInitial {
		task()
	}

	stop := make(chan struct{})
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			task()
		}
	}()

	return stop
}

// Runs a task that returns a value, said value is then marshalled and stored in the given badger DB under the given key.
// If an error occurs during the task, the error is logged and returned, and the DB write will not occur.
func PutFunc[T any](mapDB *badger.DB, key string, task func() (T, error)) (T, error) {
	dbDir := mapDB.Opts().Dir

	res, err := task()
	if err != nil {
		log.Printf("error putting '%s' into db at %s:\n%v", key, dbDir, err)
		return res, err
	}

	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("error putting '%s' into db at %s:\n%v", key, dbDir, err)
		return res, err
	}

	database.PutInsensitive(mapDB, key, data)
	log.Printf("put '%s' into db at %s\n", key, dbDir)

	return res, err
}

// func EntityMapsDifference(fresh, stale map[string]oapi.Entity) (newEntities []string, oldEntities []string) {
// 	staleNames := make(map[string]struct{})
// 	for _, e := range stale {
// 		staleNames[e.Name] = struct{}{}
// 	}

// 	freshNames := make(map[string]struct{})
// 	for _, e := range fresh {
// 		freshNames[e.Name] = struct{}{}
// 		if _, ok := staleNames[e.Name]; !ok {
// 			newEntities = append(newEntities, e.Name) // in fresh, not in stale → new
// 		}
// 	}

// 	for _, e := range stale {
// 		if _, ok := freshNames[e.Name]; !ok {
// 			oldEntities = append(oldEntities, e.Name) // in stale, not in fresh → old
// 		}
// 	}

// 	return
// }
