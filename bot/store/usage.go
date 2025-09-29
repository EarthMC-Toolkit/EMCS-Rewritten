package store

import (
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type UsageCommandStat struct {
	Name  string
	Count int
}

type UsageCommandEntry struct {
	Type      uint8 `json:"type"` // see discordgo.ApplicationCommandType
	Timestamp int64 `json:"timestamp"`
	Success   bool  `json:"success"`
}

type UserUsage struct {
	CommandHistory map[string][]UsageCommandEntry `json:"slash_command_history"` // key = command name
}

func (u *UserUsage) TotalCommandsExecuted() (total int) {
	for _, execs := range u.CommandHistory {
		total += len(execs)
	}

	return
}

// Retrieves the command entries sorted in order of most times executed first.
func (u *UserUsage) GetCommandStats() []UsageCommandStat {
	keys := slices.Collect(maps.Keys(u.CommandHistory))

	sort.Slice(keys, func(i, j int) bool {
		return len(u.CommandHistory[keys[i]]) > len(u.CommandHistory[keys[j]])
	})

	stats := make([]UsageCommandStat, 0, len(keys))
	for _, k := range keys {
		entries := u.CommandHistory[k]
		entriesLen := len(entries)

		if entriesLen > 0 {
			stats = append(stats, UsageCommandStat{
				Name:  k,
				Count: entriesLen,
				//Entry: entries[0],
			})
		}
	}

	return stats
}

func (u *UserUsage) GetCommandStatsSince(t time.Time) []UsageCommandStat {
	executionCounts := make(map[string]int) // key = command name | value = times executed since t
	for name, entries := range u.CommandHistory {
		for _, entry := range entries {
			executedAt := time.Unix(entry.Timestamp, 0).UTC()
			if executedAt.After(t) {
				executionCounts[name]++
			}
		}
	}

	stats := make([]UsageCommandStat, 0, len(executionCounts))
	for name, count := range executionCounts {
		stats = append(stats, UsageCommandStat{Name: name, Count: count})
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	return stats
}

// ================================== DATABASE INTERACTION ==================================
const USER_USAGE_KEY_PREFIX = "usage/users/"

func GetUserUsage(mapDB *badger.DB, discordID string) (*UserUsage, error) {
	return GetInsensitive[UserUsage](mapDB, USER_USAGE_KEY_PREFIX+discordID)
}

// Updates the user's usage using discordID as the key, adding entry to the history slice associated with the cmdName.
//
// All of this is done in a single transaction as opposed to two transactions (View-Get + Update-Set).
func UpdateUserUsage(mapDB *badger.DB, discordID, cmdName string, entry UsageCommandEntry) error {
	return mapDB.Update(func(txn *badger.Txn) error {
		key := []byte(strings.ToLower(USER_USAGE_KEY_PREFIX + discordID))

		item, err := txn.Get(key)
		if err != nil && err != badger.ErrKeyNotFound {
			return err
		}

		usage := UserUsage{}
		if err == nil {
			val, _ := item.ValueCopy(nil)
			if err := json.Unmarshal(val, &usage); err != nil {
				return err
			}
		} else {
			usage.CommandHistory = make(map[string][]UsageCommandEntry)
		}

		usage.CommandHistory[cmdName] = append(usage.CommandHistory[cmdName], entry)

		// marshal and set
		data, err := json.Marshal(&usage)
		if err != nil {
			return err
		}

		return txn.Set(key, data)
	})
}
