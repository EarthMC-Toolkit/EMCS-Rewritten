package slashcommands

import (
	"emcsrw/utils/config"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const MAX_THRESHOLD = 10

type DevCommand struct{}

func (cmd DevCommand) Name() string { return "dev" }
func (cmd DevCommand) Description() string {
	return "Commands for bot developers only."
}

// Subcommand "purge" will only work with the GUILD_MEMBERS intent which requires
// your bot to be verified, which you can apply for when you reach >100 servers.
func (cmd DevCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "purge",
			Description: "Leaves guilds based on low member count. Helps combat abuse.",
			Options: AppCommandOpts{
				discordutil.RequiredIntegerOption("threshold", "Guilds above this member count will not be left.", 1, MAX_THRESHOLD),
				discordutil.BoolOption("approx-only", "Determines whether to leave using only approx mem count."),
			},
		},
	}
}

func (cmd DevCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	// grab dev id from .env
	devID, err := config.GetEnviroVar("DEV_ID")
	if err != nil {
		return err
	}

	author := discordutil.GetInteractionAuthor(i.Interaction)
	if author.ID != devID {
		_, err := discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "You are not a developer silly.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	cdata := i.ApplicationCommandData()
	sub := cdata.GetOption("purge")
	threshold := int(sub.GetOption("threshold").IntValue())

	approximateOnly := false
	approxOpt := sub.GetOption("approx-only")
	if approxOpt != nil {
		approximateOnly = approxOpt.BoolValue()
	}

	guilds, err := collectAllGuilds(s)
	if err != nil {
		return err
	}
	fmt.Printf("\n/dev purge: Collected %d total guilds\n", len(guilds))

	content, _ := leaveGuilds(s, guilds, approximateOnly, threshold)
	_, err = discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return err
}

func collectAllGuilds(s *discordgo.Session) (guilds []*discordgo.UserGuild, err error) {
	after := ""
	for {
		batch, err := s.UserGuilds(200, "", after, true) // true = get approx member counts
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		guilds = append(guilds, batch...)
		after = batch[len(batch)-1].ID
	}

	return guilds, nil
}

func getHumanMemberCount(s *discordgo.Session, guildID string, threshold int) (int, error) {
	members, err := s.GuildMembers(guildID, "", threshold+1) // Uses HTTP. To use Gateway, prefer RequestGuildMembers().
	if err != nil {
		return 0, err
	}

	count := 0
	for _, m := range members {
		if !m.User.Bot {
			count++
			if count > threshold {
				return count, nil
			}
		}
	}

	return count, nil
}

// Leaves all guilds at or below the input threshold.
//
// If approx is false, both the approximate member count and real human member count are used to determine
// whether to leave a guild. Though the latter will incur more work, it should be preferred
// as the approximate count can be off by quite a bit.
func leaveGuilds(
	s *discordgo.Session, allGuilds []*discordgo.UserGuild,
	approx bool, threshold int,
) (content string, errs []error) {
	content = "Left guild(s).\n"
	leftCount, skippedCount := 0, 0

	var guildsForInspection []discordgo.UserGuild
	for _, g := range allGuilds {
		if g == nil {
			continue // corrupt guild or some shit?
		}

		// don't leave obviously massive guilds.
		// should narrow down which ones we actually need to look at
		if g.ApproximateMemberCount > MAX_THRESHOLD {
			skippedCount++
			continue
		}

		guildsForInspection = append(guildsForInspection, *g)
	}

	fmt.Printf("\nSkipped %d guilds over MAX_THRESHOLD.\n", skippedCount)

	for _, g := range guildsForInspection {
		memCount := g.ApproximateMemberCount
		if !approx {
			humans, err := getHumanMemberCount(s, g.ID, threshold)
			if err != nil {
				fmt.Print(err)
				continue
			}

			fmt.Printf("Guild %s: approx=%d, humans=%d\n", g.Name, g.ApproximateMemberCount, humans)
			memCount = humans
		}
		if memCount > threshold {
			continue // approx or real mem count still higher than threshold. dont leave
		}

		err := s.GuildLeave(g.ID)
		if err != nil {
			errs = append(errs, err)
		} else {
			content += fmt.Sprintf("- %s\n", g.Name)
			leftCount++

			fmt.Printf("Left %s [%d]\n", g.Name, memCount)
		}
	}

	if len(errs) > 0 {
		content += fmt.Sprintf("\n%d errors occurred.", len(errs))
	}

	firstLine := fmt.Sprintf("**Left %d guild(s)**.", leftCount)
	if len(content) > 2000 {
		content = firstLine
	} else {
		content = strings.Replace(content, "Left guild(s).", firstLine, 1)
	}

	return
}
