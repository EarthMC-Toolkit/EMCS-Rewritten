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
		// {
		// 	Type:        discordgo.ApplicationCommandOptionSubCommand,
		// 	Name:        "reload",
		// 	Description: "Reloads the bot by refreshing command 'Execute' definitions.",
		// },
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
	if err := discordutil.DeferReply(s, i.Interaction); err != nil {
		return err
	}

	// grab dev id from .env
	devID, err := config.GetEnviroVar("DEV_ID")
	if err != nil {
		return err
	}

	author := discordutil.GetInteractionAuthor(i.Interaction)
	if author.ID != devID {
		_, err := discordutil.EditReply(s, i.Interaction, &discordgo.InteractionResponseData{
			Content: "You are not a developer silly.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	cdata := i.ApplicationCommandData()

	// top-level sub cmd or group
	subCmd := cdata.Options[0]
	switch subCmd.Name {
	// case "reload":
	// 	return executeReload(s, i.Interaction)
	case "purge":
		return executePurge(s, i.Interaction, subCmd)
	}

	return nil
}

// func executeReload(s *discordgo.Session, i *discordgo.Interaction) error {
// 	_, err := discordutil.SendOrEditReply(s, i, &discordgo.InteractionResponseData{
// 		Content: "Creating new EMCS process and exiting current one.",
// 		Flags:   discordgo.MessageFlagsEphemeral,
// 	})

// 	reloadSelf()
// 	return err
// }

// func reloadSelf() error {
// 	executablePath, err := os.Executable()
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return err
// 	}

// 	cwd, err := os.Getwd()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Printf("\n\n---------------!! RELOAD INITIATED !!---------------\n\n")

// 	cmd := exec.Command(executablePath, "start")
// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr
// 	cmd.Stdin = os.Stdin
// 	cmd.Dir = cwd // use dir where we originally ran the bot to avoid it running in bumfuck nowhere

// 	if err := cmd.Start(); err != nil {
// 		return fmt.Errorf("failed to start bot: %w", err)
// 	}

// 	os.Exit(0)
// 	return nil
// }

func executePurge(s *discordgo.Session, i *discordgo.Interaction, subCmd *discordgo.ApplicationCommandInteractionDataOption) error {
	threshold := int(subCmd.GetOption("threshold").IntValue())

	approximateOnly := false
	approxOpt := subCmd.GetOption("approx-only")
	if approxOpt != nil {
		approximateOnly = approxOpt.BoolValue()
	}

	guilds, err := collectAllGuilds(s)
	if err != nil {
		return err
	}

	fmt.Printf("\n/dev purge: Collected %d total guilds\n", len(guilds))

	content, _ := leaveGuilds(s, guilds, approximateOnly, threshold)
	_, err = discordutil.EditReply(s, i, &discordgo.InteractionResponseData{
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
