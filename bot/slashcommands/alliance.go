package slashcommands

import (
	"emcsrw/bot/common"
	"emcsrw/bot/database"
	"emcsrw/bot/discordutil"
	"fmt"
	"math/rand"
	"time"

	"github.com/bwmarrin/discordgo"
)

type AllianceCommand struct{}

func (cmd AllianceCommand) Name() string { return "alliance" }
func (cmd AllianceCommand) Description() string {
	return "Look up and alliance or request one be created/edited."
}

func (cmd AllianceCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about an alliance (meganation or pact between nations).",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance.",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The short unique name used to look up the alliance.", 3, 16),
				discordutil.RequiredStringOption("label", "The full name for display purposes.", 4, 36),
			},
		},
	}
}

func (cmd AllianceCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	err := discordutil.DeferReply(s, i.Interaction)
	if err != nil {
		return err
	}

	cmdData := i.ApplicationCommandData()
	if lookup := cmdData.GetOption("query"); lookup != nil {
		return QueryAlliance(s, i.Interaction, cmdData)
	}
	if create := cmdData.GetOption("create"); create != nil {
		return CreateAlliance(s, i.Interaction, cmdData)
	}

	return err
}

func QueryAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
	opt := data.GetOption("query")
	ident := opt.GetOption("identifier").StringValue()

	// Try find alliance in DB
	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	alliance, err := database.GetAllianceByIdentifier(db, ident)
	if err != nil {
		fmt.Printf("failed to get alliance '%s' from db: %v", ident, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	_, err = discordutil.FollowUpEmbeds(s, i, common.NewAllianceEmbed(s, alliance))
	return err
}

func CreateAlliance(s *discordgo.Session, i *discordgo.Interaction, data discordgo.ApplicationCommandInteractionData) error {
	opt := data.GetOption("create")

	ident := opt.GetOption("identifier").StringValue()
	label := opt.GetOption("label").StringValue()

	createdAlliance := &database.Alliance{
		ID:         generateAllianceID(),
		Identifier: ident,
		Label:      label,
	}

	db := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	err := database.PutAlliance(db, createdAlliance)
	if err != nil {
		fmt.Printf("failed to put alliance %s into db:\n%v", ident, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not create alliance `%s`! Check the console.", ident))
		return err
	}

	_, err = discordutil.FollowUpContent(s, i, fmt.Sprintf("Successfully created alliance `%s (%s)`", label, ident))
	return err
}

func generateAllianceID() uint64 {
	created := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))      // Safe to cast to uint since Intn returns 0-n anyway.
	return (created << 16) | suffix
}
