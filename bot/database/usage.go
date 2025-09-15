package database

import (
	"encoding/json"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/dgraph-io/badger/v4"
)

type CommandStat struct {
	Name  string
	Entry CommandEntry
	Count int
}

type CommandEntry struct {
	Type      uint8 `json:"type"` // see discordgo.ApplicationCommandType
	Timestamp int64 `json:"timestamp"`
	Success   bool  `json:"success"`
}

type UserUsage struct {
	CommandHistory map[string][]CommandEntry `json:"slash_command_history"` // key = command name
}

func (u *UserUsage) TotalCommandsExecuted() (total int) {
	for _, execs := range u.CommandHistory {
		total += len(execs)
	}

	return
}

// Retrieves the command entries sorted in order of most times executed first.
func (u *UserUsage) GetCommandStats() []CommandStat {
	keys := slices.Collect(maps.Keys(u.CommandHistory))

	sort.Slice(keys, func(i, j int) bool {
		return len(u.CommandHistory[keys[i]]) > len(u.CommandHistory[keys[j]])
	})

	stats := make([]CommandStat, 0, len(keys))
	for _, k := range keys {
		entries := u.CommandHistory[k]
		if len(entries) > 0 {
			stats = append(stats, CommandStat{
				Name:  k,
				Entry: entries[0],
				Count: len(entries),
			})
		}
	}

	return stats
}

// ================================== DATABASE INTERACTION ==================================
const USER_USAGE_KEY_PREFIX = "usage/users/"

func GetUserUsage(mapDB *badger.DB, discordID string) (*UserUsage, error) {
	return GetInsensitive[UserUsage](mapDB, USER_USAGE_KEY_PREFIX+discordID)
}

// func PutUserUsage(mapDB *badger.DB, u *UserUsage, discordID string) error {
// 	data, err := json.Marshal(u)
// 	if err != nil {
// 		return fmt.Errorf("error putting user '%s' usage into db at %s:\n%v", discordID, mapDB.Opts().Dir, err)
// 	}

// 	return PutInsensitive(mapDB, USER_USAGE_KEY_PREFIX+discordID, data)
// }

// Updates the user's usage using discordID as the key, adding entry to the history slice associated with the cmdName.
//
// All of this is done in a single transaction as opposed to two transactions (View-Get + Update-Set).
func UpdateUserUsage(mapDB *badger.DB, discordID, cmdName string, entry CommandEntry) error {
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
			usage.CommandHistory = make(map[string][]CommandEntry)
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
