package events

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/discordutil"
	"emcsrw/bot/slashcommands"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/bwmarrin/discordgo"
)

func OnInteractionCreateApplicationCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler InteractionCreateApplicationCommand recovered from a panic.\n%v\n%s", err, debug.Stack())

			errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
			content := "Bot attempted to panic during this command. Please report the following error.\n" + errStr

			// Not already deferred, reply.
			err := discordutil.SendReply(s, i.Interaction, &discordgo.InteractionResponseData{
				Flags:   discordgo.MessageFlagsEphemeral,
				Content: content,
			})

			if err != nil {
				// Must be deferred, send follow up.
				discordutil.FollowUpContentEphemeral(s, i.Interaction, content)
			}
		}
	}()

	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	author := discordutil.UserFromInteraction(i.Interaction)

	cmdName := i.ApplicationCommandData().Name
	cmdType := i.ApplicationCommandData().CommandType
	cmd := slashcommands.All()[cmdName]

	start := time.Now()
	err := cmd.Execute(s, i)
	elapsed := time.Since(start)

	success := false
	if err != nil {
		fmt.Printf("\n'%s' failed to execute command /%s:\n%v", author.Username, cmdName, err)
	} else {
		fmt.Printf("\n'%s' successfully executed command /%s (took: %s)\n", author.Username, cmdName, elapsed)
		success = true
	}

	// TODO: Maybe combine get/put into single update transaction. Rn this is View&Get + Update&Set
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	// usage, err := database.GetUserUsage(db, author.ID)
	// if err != nil {
	// 	fmt.Printf("\ndb error occurred. could not get usage for user: %s (%s)\n%v", author.Username, author.ID, err)
	// 	return
	// }

	// Add a new command usage entry to history.
	// usage.CommandHistory[cmdName] = append(usage.CommandHistory[cmdName], database.CommandEntry{
	// 	Type:      uint8(cmdType),
	// 	Timestamp: time.Now().Unix(),
	// 	Success:   success,
	// })

	if cmdName != "usage" {
		err = database.UpdateUserUsage(db, author.ID, cmdName, database.CommandEntry{
			Type:      uint8(cmdType),
			Timestamp: time.Now().Unix(),
			Success:   success,
		})
	}

	if err != nil {
		fmt.Printf("\ndb error occurred. could not update usage for user: %s (%s)\n%v", author.Username, author.ID, err)
	}
}

// List of message components: https://discord.com/developers/docs/components/reference#component-object-component-types
func OnInteractionCreateMessageComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

}
