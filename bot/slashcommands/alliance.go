package slashcommands

import (
	"emcsrw/api/oapi"
	"emcsrw/bot/store"
	"emcsrw/shared"
	"emcsrw/utils/discordutil"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/samber/lo"
)

const EDITOR_ROLE = "966359842417705020"
const SR_EDITOR_ROLE = "1143253762039873646"

type AllianceCommand struct{}

func (cmd AllianceCommand) Name() string { return "alliance" }
func (cmd AllianceCommand) Description() string {
	return "Look up and alliance or request one be created/edited."
}

func (cmd AllianceCommand) Options() AppCommandOpts {
	return AppCommandOpts{
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "create",
			Description: "Create an alliance. This command is for Editors only.",
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
			Name:        "edit",
			Description: "Edit alliance info. This command is for Editors only.",
			Options: AppCommandOpts{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "functional",
					Description: "Edit alliance fields that are required for basic functionality.",
					Options: AppCommandOpts{
						discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "optional",
					Description: "Edit alliance fields that are not tied to its functionality.",
					Options: AppCommandOpts{
						discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
					},
				},
			},
		},
		{
			Type:        discordgo.ApplicationCommandOptionSubCommand,
			Name:        "query",
			Description: "Query information about an alliance (meganation, organisation or pact).",
			Options: AppCommandOpts{
				discordutil.RequiredStringOption("identifier", "The alliance's identifier/short name.", 3, 16),
			},
		},
	}
}

func (cmd AllianceCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	cdata := i.ApplicationCommandData()

	opt := cdata.GetOption("query")
	if opt != nil {
		err := discordutil.DeferReply(s, i.Interaction)
		if err != nil {
			return err
		}

		return queryAlliance(s, i.Interaction, cdata)
	}

	if opt = cdata.GetOption("create"); opt != nil {
		return createAlliance(s, i.Interaction)
	}
	if opt = cdata.GetOption("edit"); opt != nil {
		return editAlliance(s, i.Interaction, cdata)
	}

	return nil
}

func queryAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	ident := cdata.GetOption("query").GetOption("identifier").StringValue()
	alliance, err := allianceStore.GetKey(strings.ToLower(ident))
	if err != nil {
		//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)

		_, err := discordutil.FollowUpContentEphemeral(s, i, fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident))
		return err
	}

	_, err = discordutil.FollowUpEmbeds(s, i, shared.NewAllianceEmbed(s, alliance))
	return err
}

func createAlliance(s *discordgo.Session, i *discordgo.Interaction) error {
	isEditor, _ := discordutil.HasRole(i.Member, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	// See handleAllianceCreatorModal() for submission handling,
	// where the actual creation and saving to the database occurs.
	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_creator",
		Title:    "Alliance Creator",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "identifier",
				Label:       "Query Identifier (3-16 chars)",
				Placeholder: "Enter a unique short name used to query this alliance.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   3,
				MaxLength:   16,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "label",
				Label:       "Alliance Name (4-36 chars)",
				Placeholder: "Enter this alliance's full name.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "representative",
				Label:       "Representative Discord ID",
				Placeholder: "Enter the Discord ID of the user representing this alliance.",
				Required:    true,
				Style:       discordgo.TextInputShort,
				MinLength:   17,
				MaxLength:   19,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "nations",
				Label:       "Own Nations",
				Placeholder: "Enter a comma-seperated list of nations in THIS alliance only.",
				Required:    true,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "parents",
				Label:       "Parent Alliance(s)",
				Placeholder: "(Optional) Enter a comma-seperated list of alliance *identifiers* which this alliance is a child of.",
				Required:    false,
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
		},
	})
}

func editAlliance(s *discordgo.Session, i *discordgo.Interaction, cdata discordgo.ApplicationCommandInteractionData) error {
	isEditor, _ := discordutil.HasRole(i.Member, EDITOR_ROLE)
	if !isEditor && !discordutil.IsDev(i) {
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: "Stop trying.",
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	allianceStore, err := store.GetStoreForMap[store.Alliance](shared.ACTIVE_MAP, "alliances")
	if err != nil {
		return err
	}

	opt := cdata.GetOption("edit")
	functional := opt.GetOption("functional")
	optional := opt.GetOption("optional")
	if functional != nil {
		opt = functional
	} else {
		opt = optional
	}

	ident := opt.GetOption("identifier").StringValue()
	alliance, err := allianceStore.GetKey(strings.ToLower(ident))
	if err != nil {
		//fmt.Printf("failed to get alliance by identifier '%s' from db: %v", ident, err)
		_, err := discordutil.EditOrSendReply(s, i, &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Could not find alliance by identifier: `%s`.", ident),
			Flags:   discordgo.MessageFlagsEphemeral,
		})

		return err
	}

	if functional != nil {
		return openEditorModalFunctional(s, i, alliance)
	}

	return openEditorModalOptional(s, i, alliance)
}

func openEditorModalFunctional(s *discordgo.Session, i *discordgo.Interaction, alliance *store.Alliance) error {
	nationStore, _ := store.GetStoreForMap[oapi.NationInfo](shared.ACTIVE_MAP, "nations")
	nations := alliance.GetNations(nationStore)
	nationNames := lo.Map(nations, func(n oapi.NationInfo, _ int) string {
		return n.Name
	})

	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_functional@" + alliance.Identifier,
		Title:    "Alliance Editor - Functional Fields",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "identifier",
				Label:       "Query Identifier (3-16 chars)",
				Placeholder: alliance.Identifier,
				Style:       discordgo.TextInputShort,
				MinLength:   3,
				MaxLength:   16,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "label",
				Label:       "Alliance Name (4-36 chars)",
				Placeholder: alliance.Label,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   36,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "representative",
				Label:       "Representative Discord ID",
				Placeholder: *alliance.RepresentativeID,
				Style:       discordgo.TextInputShort,
				MinLength:   17,
				MaxLength:   19,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "nations",
				Label:       "Own Nations",
				Placeholder: strings.Join(nationNames, ", "),
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "parents",
				Label:       "Parent Alliance(s)",
				Placeholder: strings.Join(alliance.Parents, ", "),
				MinLength:   3,
				Style:       discordgo.TextInputParagraph,
			}),
		},
	})
}

func openEditorModalOptional(s *discordgo.Session, i *discordgo.Interaction, alliance *store.Alliance) error {
	discordPlaceholder := "Enter an invite link or code to the alliance's Discord."
	if alliance.Optional.DiscordCode != nil {
		discordPlaceholder = fmt.Sprintf("https://discord.gg/%s", *alliance.Optional.DiscordCode)
	}

	imagePlaceholder := "Enter the URL of the alliance's image/flag from the flags channel."
	if alliance.Optional.ImageURL != nil {
		imagePlaceholder = *alliance.Optional.ImageURL
	}

	leaderPlaceholder := "Enter the Minecraft IGNs of the alliance leaders, comma-separated."
	if alliance.Optional.Leaders != nil {
		leaderNames, err := alliance.GetLeaderNames()
		if err == nil {
			leaderPlaceholder = strings.Join(leaderNames, ", ")
		}
	}

	coloursPlaceholder := "Enter HEX colour(s) seperated by a space. Fill first, Outline second."

	colours := alliance.Optional.Colours
	if colours != nil && colours.Fill != nil {
		fill := strings.TrimPrefix(*colours.Fill, "#")
		coloursPlaceholder = "#" + fill

		if colours.Outline != nil {
			coloursPlaceholder += " #" + strings.TrimPrefix(*colours.Outline, "#")
		} else {
			coloursPlaceholder += " #" + fill
		}
	}

	return discordutil.OpenModal(s, i, &discordgo.InteractionResponseData{
		CustomID: "alliance_editor_optional@" + alliance.Identifier,
		Title:    "Alliance Editor - Optional Fields",
		Components: []discordgo.MessageComponent{
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "type",
				Label:       "Alliance Type (mega/org/pact)",
				Placeholder: string(alliance.Type),
				MinLength:   3,
				MaxLength:   4,
				Style:       discordgo.TextInputShort,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "discord",
				Label:       "Permanent Discord Invite",
				Placeholder: discordPlaceholder,
				Style:       discordgo.TextInputShort,
				MinLength:   4,
				MaxLength:   40,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "image",
				Label:       "Image/Flag URL",
				Placeholder: imagePlaceholder,
				Style:       discordgo.TextInputShort,
				MinLength:   10,
				MaxLength:   100,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "colours",
				Label:       "Colours (Used by bot & extension)",
				Placeholder: coloursPlaceholder,
				MinLength:   4,
				Style:       discordgo.TextInputShort,
			}),
			discordutil.TextInputActionRow(discordgo.TextInput{
				CustomID:    "leaders",
				Label:       "Leader IGNs (comma-separated)",
				Placeholder: leaderPlaceholder,
				Style:       discordgo.TextInputParagraph,
				MinLength:   3,
				MaxLength:   320, // Minecraft max name length is 16. Should suffice for many leaders.
			}),
		},
	})
}
