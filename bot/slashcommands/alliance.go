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

func RequiredStringOption(name, description string, minLen, maxLen int) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        name,
		Description: description,
		MinLength:   &minLen,
		MaxLength:   maxLen,
		Required:    true,
	}
}

type AllianceCommand struct{}

func (cmd AllianceCommand) Name() string { return "alliance" }
func (cmd AllianceCommand) Description() string {
	return "Look up and alliance or request one be created/edited."
}

func (cmd AllianceCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "lookup",
			Description: "Retrieves information about an alliance via it's identifier.",
			Options: AppCommandOpts{
				RequiredStringOption("name", "The alliance's identifier/short name.", 3, 16),
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance.",
			Options: AppCommandOpts{
				RequiredStringOption("identifier", "The short unique name used to look up the alliance.", 3, 16),
				RequiredStringOption("label", "The full name for display purposes.", 4, 36),
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

	lookup := cmdData.GetOption("lookup")
	if lookup != nil {
		return LookupAlliance(s, i.Interaction, cmdData)
	}

	create := cmdData.GetOption("create")
	if create != nil {
		return CreateAlliance(s, i.Interaction, cmdData)
	}

	return err
}

func LookupAlliance(s *discordgo.Session, i *discordgo.Interaction, cmdData discordgo.ApplicationCommandInteractionData) error {
	// TODO	1. Grab alliance from db
	// 		2. Create embed from alliance info
	// 		3. Send embed with followup

	return nil
}

func CreateAlliance(s *discordgo.Session, i *discordgo.Interaction, cmdData discordgo.ApplicationCommandInteractionData) error {
	identifier := cmdData.GetOption("identifier").StringValue()
	label := cmdData.GetOption("label").StringValue()

	createdAlliance := &database.Alliance{
		ID:         generateAllianceID(),
		Identifier: identifier,
		Label:      label,
	}

	auroraDB := database.GetMapDB(common.SUPPORTED_MAPS.AURORA)
	err := database.PutAlliance(auroraDB, createdAlliance)
	if err != nil {
		fmt.Printf("failed to put alliance %s into db:\n%v", identifier, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, "could not create alliance! check the console")
		return err
	}

	_, err = discordutil.FollowUpContent(s, i, fmt.Sprintf("successfully created alliance `%s (%s)`", label, identifier))
	return err
}

func generateAllianceID() uint64 {
	created := uint64(time.Now().UnixMilli()) // Shouldn't ever be negative after 1970 :P
	suffix := uint64(rand.Intn(1 << 16))      // Safe to cast to uint since Intn returns 0-n anyway.
	return (created << 16) | suffix
}
