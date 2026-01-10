package slashcommands

import (
	"emcsrw/utils/config"
	"emcsrw/utils/discordutil"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const MAX_THRESHOLD = 10

type DevCommand struct{}

func (cmd DevCommand) Name() string { return "dev" }
func (cmd DevCommand) Description() string {
	return "Commands for bot developers only."
}

func (cmd DevCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "purge",
			Description: "Leaves guilds based on low member count. Helps combat abuse.",
			Options: AppCommandOpts{
				discordutil.RequiredIntegerOption("threshold", "Guilds above this member count will not be left.", 1, MAX_THRESHOLD),
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

	guilds, err := collectAllGuilds(s)
	if err != nil {
		return err
	}

	fmt.Printf("\n/dev purge: Collected %d total guilds\n", len(guilds))

	content := "**Left guild(s)**:\n"
	errs := []error{}

	leftCount := 0

	// leave all guilds at or below input threshold
	for _, g := range guilds {
		// don't leave obviously massive guilds.
		// should narrow narrow down which ones we actually need to look at
		if g.ApproximateMemberCount > threshold {
			continue
		}

		fmt.Printf("Left %s [%d]\n", g.Name, g.ApproximateMemberCount)
		err = s.GuildLeave(g.ID)
		if err != nil {
			errs = append(errs, err)
		} else {
			content += fmt.Sprintf("- %s\n", g.Name)
			leftCount++
		}
	}

	if len(errs) > 0 {
		content += fmt.Sprintf("\nThe following errors occurred:\n%s", errors.Join(errs...))
	}

	if len(content) >= 4096 {
		content = fmt.Sprintf("Left %d guild(s).", leftCount)
	}

	_, err = discordutil.EditOrSendReply(s, i.Interaction, &discordgo.InteractionResponseData{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})

	return err
}

func collectAllGuilds(s *discordgo.Session) ([]*discordgo.UserGuild, error) {
	allGuilds := []*discordgo.UserGuild{}
	after := ""

	for {
		batch, err := s.UserGuilds(200, "", after, true) // true = get approx member counts
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}

		allGuilds = append(allGuilds, batch...)
		after = batch[len(batch)-1].ID
	}

	return allGuilds, nil
}

// func getHumanMemberCount(s *discordgo.Session, guildID string, threshold int) (int, error) {
// 	count, limit := 0, threshold+1

// 	// Fetch a single batch; no loop needed
// 	members, err := s.GuildMembers(guildID, "", limit)
// 	if err != nil {
// 		return 0, err
// 	}

// 	for _, m := range members {
// 		if !m.User.Bot {
// 			count++
// 			if count > threshold {
// 				return count, nil
// 			}
// 		}
// 	}

// 	return count, nil
// }
