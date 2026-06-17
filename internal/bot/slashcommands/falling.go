package slashcommands

import (
	"emcsrw/internal/database"
	"emcsrw/internal/shared"
	"emcsrw/pkg/utils"
	"emcsrw/pkg/utils/discordutil"
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

var fallingTownSorts = map[string]func([]database.FallingTown){
	"alphabetical": func(towns []database.FallingTown) {
		utils.KeySort(towns, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool { return a.Name < b.Name }},
		})
	},
	"residents": func(towns []database.FallingTown) {
		utils.KeySort(towns, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool { return a.NumResidents() > b.NumResidents() }},
		})
	},
	"size": func(towns []database.FallingTown) {
		utils.KeySort(towns, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool { return a.Size() > b.Size() }},
		})
	},
	"founded": func(towns []database.FallingTown) {
		utils.KeySort(towns, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool {
				return a.Timestamps.Registered < b.Timestamps.Registered
			}},
		})
	},
	"balance": func(towns []database.FallingTown) {
		utils.KeySort(towns, []utils.KeySortOption[database.FallingTown]{
			{Compare: func(a, b database.FallingTown) bool { return a.Bal() > b.Bal() }},
		})
	},
	"overclaimed": func(towns []database.FallingTown) {
		utils.RankSortAscending(towns, func(t database.FallingTown) int {
			switch {
			case t.Status.Overclaimed:
				return 0
			case t.Status.Overclaimed:
				return 1
			default:
				return 2
			}
		})
	},
	"has-nation": func(towns []database.FallingTown) {
		utils.SortToggledOn(towns, func(t database.FallingTown) bool {
			return t.Status.HasNation
		})
	},
	"can-outsiders-spawn": func(towns []database.FallingTown) {
		utils.SortToggledOn(towns, func(t database.FallingTown) bool {
			return t.Status.CanOutsidersSpawn
		})
	},
	"open": func(towns []database.FallingTown) {
		utils.SortToggledOn(towns, func(t database.FallingTown) bool {
			return t.Status.Open
		})
	},
	"public": func(towns []database.FallingTown) {
		utils.SortToggledOn(towns, func(t database.FallingTown) bool {
			return t.Status.Public
		})
	},
	"neutral": func(towns []database.FallingTown) {
		utils.SortToggledOn(towns, func(t database.FallingTown) bool {
			return t.Status.Neutral
		})
	},
}

// TODO: Add a rate limiter to this command to stop the token bucket being empty.
// Keep a record of time last used per user, and allow again after X minutes.
type FallingCommand struct {
	//lastUsed map[string]int64
}

func (cmd FallingCommand) Name() string { return "falling" }
func (cmd FallingCommand) Description() string {
	return "Sends a list of towns about to fall into ruin within 7 days. Includes stats, status etc."
}

func (cmd FallingCommand) Options() []AppCommandOpt {
	return []AppCommandOpt{
		discordutil.StringOption("sort", "Optional town list sorting. Without this, towns are sorted by residents -> size.", nil, nil,
			discordutil.Choice("Alphabetical", "alphabetical"), // "Sort the list alphabetically by name."
			discordutil.Choice("Founded", "founded"),           // "Sort the list by date founded. Oldest -> Newest."
			discordutil.Choice("Residents", "residents"),       // "Sort the list solely by the number of residents."
			discordutil.Choice("Size", "size"),                 // "Sort the list solely by size (chunks claimed)."
			discordutil.Choice("Balance", "balance"),           // "Sort the list solely by balance."
			discordutil.Choice("Ruined", "ruined"),             // "Sort the list by for sale status. For Sale (highest-lowest) -> Not For Sale."
			discordutil.Choice("Overclaimed", "overclaimed"),   // "Sort the list by overclaim status. Oldest -> Newest."
			discordutil.Choice("For Sale", "for-sale"),         // "Sort the list by for sale status. For Sale (highest-lowest) -> Not For Sale."
			discordutil.Choice("Has Nation", "has-nation"),
			discordutil.Choice("Can Outsiders Spawn", "can-outsiders-spawn"), // "Sort the list by outsider spawn status. Enabled -> Not enabled."
			discordutil.Choice("Open", "open"),                               // "Sort the list by open status. Open -> Not open."
			discordutil.Choice("Public", "public"),                           // "Sort the list by public status. Public -> Not public."
			discordutil.Choice("Neutral", "neutral"),                         // "Sort the list by neutral status. Neutral -> Not neutral."
		),
		discordutil.AutocompleteStringOption("nation", "Filter by towns within a specified nation.", 2, 40, false),
		discordutil.StringOption("status", "Filter by towns with the specified status active.", nil, nil,
			discordutil.Choice("Ruined", "ruined"),
			discordutil.Choice("Overclaimed", "overclaimed"),
			discordutil.Choice("For Sale", "for-sale"),
			discordutil.Choice("Can Outsiders Spawn", "can-outsiders-spawn"),
			discordutil.Choice("Open", "open"),
			discordutil.Choice("Public", "public"),
			discordutil.Choice("Neutral", "neutral"),
		),
		discordutil.StringOption("flags", "Filter by towns with the specified flag toggled on.", nil, nil,
			discordutil.Choice("Explosions", "explosions"),
			discordutil.Choice("Mobs", "mobs"),
			discordutil.Choice("Fire", "fire"),
			discordutil.Choice("PVP", "pvp"),
		),
	}
}

func (cmd FallingCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	msg := discordutil.NewMessageBuilder()
	msg.SetContent(shared.EMOJIS.LOADING + " This command may take a short while, please wait...")
	if err := discordutil.SendReply(s, i.Interaction, msg.InteractionData()); err != nil {
		return err
	}

	mdb, err := database.Get(shared.ACTIVE_MAP)
	if err != nil {
		return err
	}

	fallingTownsStore, err := database.GetStore(mdb, database.FALLING_TOWNS_STORE)
	if err != nil {
		return err
	}

	falling := fallingTownsStore.Values()

	cdata := i.ApplicationCommandData()
	if opt := cdata.GetOption("nation"); opt != nil {
		nationInput := opt.StringValue()

		// parse input so "Japan" and "Japan (Capital: Chiyoda)" both work by extracting the nation name before any parentheses,
		// since that's what the autocomplete displays and it seems reasonable to expect users to copy-paste from autocomplete results.
		// "((Japan))", "(Japan (blahblah))" etc. also need to work incase of stupid nation names.
		nationName := strings.TrimSpace(nationInput)
		for strings.HasPrefix(nationName, "(") && strings.HasSuffix(nationName, ")") {
			nationName = strings.TrimSpace(nationName[1 : len(nationName)-1])
		}
		if i := strings.IndexRune(nationName, '('); i >= 0 {
			nationName = strings.TrimSpace(nationName[:i])
		}

		falling = lo.Filter(falling, func(t database.FallingTown, _ int) bool {
			if t.Nation.UUID == nil {
				return false
			}

			return strings.EqualFold(*t.Nation.Name, nationName)
		})
	}

	// Apply custom filter if chosen
	if opt := cdata.GetOption("status"); opt != nil {
		if filter, ok := statusFilters[opt.StringValue()]; ok {
			falling = lo.Filter(falling, func(t database.FallingTown, _ int) bool {
				return filter(t.TownInfo)
			})
		}
	}
	if opt := cdata.GetOption("flags"); opt != nil {
		if filter, ok := flagFilters[opt.StringValue()]; ok {
			falling = lo.Filter(falling, func(t database.FallingTown, _ int) bool {
				return filter(t.TownInfo)
			})
		}
	}

	// Apply custom sort if chosen
	if opt := cdata.GetOption("sort"); opt != nil {
		if sorter, ok := fallingTownSorts[opt.StringValue()]; ok {
			sorter(falling)
		}
	} else {
		// Default sort (least active mayor first)
		sort.Slice(falling, func(i, j int) bool {
			if falling[i].InactiveDuration == falling[j].InactiveDuration {
				return falling[i].TownInfo.NumResidents() < falling[j].TownInfo.NumResidents()
			}
			return falling[i].InactiveDuration > falling[j].InactiveDuration
		})
	}

	totalCount := len(falling)

	perPage := 5
	paginator := discordutil.NewInteractionPaginator(s, i.Interaction, totalCount, perPage).
		WithTimeout(10 * time.Minute)

	paginator.PageFunc = func(curPage int, data *discordgo.InteractionResponseData) {
		start, end := paginator.CurrentPageBounds(totalCount)
		items := falling[start:end]

		descBuilder := strings.Builder{} // More performant than concat via regular Sprintf
		for idx, t := range items {
			nationName := "No Nation"
			if t.Nation.Name != nil {
				nationName = *t.Nation.Name
			}

			X, Y, Z := t.SpawnLocation()
			locationLink := fmt.Sprintf("[%.0f, %.0f, %.0f](https://map.earthmc.net?x=%f&z=%f&zoom=6)", X, Y, Z, X, Z)

			residents := logutil.HumanizedSprintf("%s `%d`", shared.EMOJIS.RESIDENT_PURPLE, t.NumResidents())
			balance := logutil.HumanizedSprintf("%s `%.0f` ", shared.EMOJIS.GOLD_INGOT, t.Bal())
			chunks := logutil.HumanizedSprintf("%s `%d` ", shared.EMOJIS.CHUNK, t.Size())

			emojis := shared.EMOJIS
			capital := fmt.Sprintf("%s Capital", lo.Ternary(t.Status.Capital, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			open := fmt.Sprintf("%s Open", lo.Ternary(t.Status.Open, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			spawn := fmt.Sprintf("%s Outsider Spawn", lo.Ternary(t.Status.CanOutsidersSpawn, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))
			pvp := fmt.Sprintf("%s PVP", lo.Ternary(t.Perms.Flags.PVP, emojis.CIRCLE_CHECK, emojis.CIRCLE_CROSS))

			fmt.Fprintf(&descBuilder, "%d. **%s** (%s) will fall <t:%d:R> at %s.\n"+
				"Deletion on `%s` (<t:%d:R>).\n"+
				"Mayor: `%s` (online <t:%d:R>). %s • %s • %s\n"+
				"%s %s %s %s\n\n",
				start+idx+1, t.Name, nationName, t.RuinAt.Unix(), locationLink, // line 1
				utils.FormatTime(t.DeletionAt), t.DeletionAt.Unix(), // line 2
				t.Mayor.Name, t.MayorLastOnline.Unix(), residents, balance, chunks, // line 3
				capital, open, spawn, pvp, // line 4
			)
		}

		pageStr := fmt.Sprintf("Page %d/%d", curPage+1, paginator.TotalPages())
		title := fmt.Sprintf("[%d] List of Falling Towns | %s", totalCount, pageStr)
		desc := descBuilder.String()

		embed := discordutil.NewEmbedBuilder(&discordutil.YELLOW, &title, &desc, nil)
		data.Embeds = []*discordgo.MessageEmbed{embed.Build()}
	}

	return paginator.Start()
}
