package database

import (
	"maps"
	"slices"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
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

// A version of CalculateTotal that does not require an input slice since
// we can just use the command history length directly.
func (u UserUsage) TotalCommandsExecuted() (total int) {
	for _, execs := range u.CommandHistory {
		total += len(execs)
	}

	return
}

func (u UserUsage) CalculateTotal(stats []UsageCommandStat) (total int) {
	for _, s := range stats {
		total += s.Count
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

func UpdateUsageForUser(db *Database, user *discordgo.User, cmdName string, entry UsageCommandEntry) error {
	usageStore, err := GetStore(db, USAGE_USERS_STORE)
	if err != nil {
		return err
	}

	usage, _ := usageStore.Get(user.ID)
	if usage == nil {
		usage = &UserUsage{
			CommandHistory: make(map[string][]UsageCommandEntry),
		}
	}

	usage.CommandHistory[cmdName] = append(usage.CommandHistory[cmdName], entry)
	usageStore.Set(user.ID, *usage)

	return nil
}
