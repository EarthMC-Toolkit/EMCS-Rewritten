package discordutil

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

const AUTOCOMPLETE_CHOICE_LIMIT = 25

func OpenModal(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: data,
	})
}

func DeferComponent(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})
}

// Responds to an interaction with a deferred response, allowing more time to process before sending a follow-up message.
//
// Deferred interactions cannot carry data and can only be edited or followed up.
func DeferReply(s *discordgo.Session, i *discordgo.Interaction) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func SendReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) error {
	return s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: data,
	})
}

func EditReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
	return s.InteractionResponseEdit(i, &discordgo.WebhookEdit{
		Content:         &data.Content,
		Embeds:          &data.Embeds,
		Files:           data.Files,
		Components:      &data.Components,
		AllowedMentions: data.AllowedMentions,
	})
}

func EditOrSendReply(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData) (*discordgo.Message, error) {
	if err := SendReply(s, i, data); err == nil {
		return nil, nil
	}

	return EditReply(s, i, data)
}

// func SendOrFollowup(s *discordgo.Session, i *discordgo.Interaction, data *discordgo.InteractionResponseData, ephemeral bool) (*discordgo.Message, error) {
// 	// Try sending the initial reply
// 	err := s.InteractionRespond(i, &discordgo.InteractionResponse{
// 		Type: discordgo.InteractionResponseChannelMessageWithSource,
// 		Data: data,
// 	})
// 	if err == nil {
// 		return nil, nil // initial reply sent successfully
// 	}

// 	// If it failed because a response already exists, send a follow-up
// 	return s.FollowupMessageCreate(i, ephemeral, &discordgo.WebhookParams{
// 		Content:         data.Content,
// 		Embeds:          data.Embeds,
// 		Files:           data.Files,
// 		Components:      data.Components,
// 		AllowedMentions: data.AllowedMentions,
// 	})
// }

func ReplyWithError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	// NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := fmt.Sprintf("Bot encountered a non-fatal error during this command.```%s```", err)

	// Not already deferred, reply.
	_, err = EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		FollowupContentEphemeral(s, i, content)
	}
}

func ReplyWithPanicError(s *discordgo.Session, i *discordgo.Interaction, err any) {
	errStr := fmt.Sprintf("```%v```", err) // NOTE: This could panic itself. Maybe handle it or just send generic text.
	content := "Bot attempted to fatally crash during this command! Please report the following error.\n" + errStr

	// Not already deferred, reply.
	_, err = EditOrSendReply(s, i, &discordgo.InteractionResponseData{
		Flags:   discordgo.MessageFlagsEphemeral,
		Content: content,
	})

	if err != nil {
		// Must be deferred, send follow up.
		FollowupContentEphemeral(s, i, content)
	}
}

// Creates a follow-up message for a previously deferred interaction response.
// This func waits for server confirmation of message send and ensures that the return struct is populated.
func Followup(s *discordgo.Session, i *discordgo.Interaction, params *discordgo.WebhookParams) (*discordgo.Message, error) {
	return s.FollowupMessageCreate(i, true, params)
}

// Calls FollowUp with the supplied embeds.
func FollowupEmbeds(s *discordgo.Session, i *discordgo.Interaction, embeds ...*discordgo.MessageEmbed) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Embeds: embeds,
	})
}

// Calls FollowUp with the supplied content.
func FollowupContent(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Content: content,
	})
}

// Calls FollowUp with the supplied content which will only be visible to the interaction author.
func FollowupContentEphemeral(s *discordgo.Session, i *discordgo.Interaction, content string) (*discordgo.Message, error) {
	return Followup(s, i, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
}

// Attempts to get the username from an interaction.
//
// Regular `User` is only filled for a DM, so this func uses guild-specific `Member.User` otherwise.
func GetInteractionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.User != nil {
		return i.User
	}

	return i.Member.User
}

// This function attemps to find and save the value of every TextInput on the components array of the modal submit data.
//
// In Message Components V2, it will go through the component of every Label, for legacy components (V2),
// it will look for it within an ActionsRow. The input values are then stored in map keyed by CustomID of the TextInput.
func GetModalInputs(i *discordgo.Interaction) map[string]string {
	if i.Type != discordgo.InteractionModalSubmit {
		// in case some retard (me) tries to use this func to get inputs for different command type
		panic("failed to get modal inputs: expected interaction type InteractionModalSubmit")
	}

	inputs := make(map[string]string)
	for _, modalComp := range i.ModalSubmitData().Components {
		switch modalComp.Type() {
		case discordgo.LabelComponent:
			if input, ok := modalComp.(*discordgo.Label).Component.(*discordgo.TextInput); ok {
				inputs[input.CustomID] = input.Value
			}
		case discordgo.ActionsRowComponent:
			// Gather values of all text input components in this row.
			for _, comp := range modalComp.(*discordgo.ActionsRow).Components {
				if input, ok := comp.(*discordgo.TextInput); ok {
					inputs[input.CustomID] = input.Value
				}
			}
		}
	}

	return inputs
}

func GetFocusedValue[T any](options []*discordgo.ApplicationCommandInteractionDataOption) (v T, ok bool) {
	for _, opt := range options {
		if opt.Focused {
			v, ok = opt.Value.(T)
			return
		}
		if len(opt.Options) > 0 {
			if v, ok = GetFocusedValue[T](opt.Options); ok {
				return
			}
		}
	}

	return
}

func traverseCommandOption(opt *discordgo.ApplicationCommandInteractionDataOption) *discordgo.ApplicationCommandInteractionDataOption {
	if len(opt.Options) == 0 {
		return opt
	}

	// Check if the first child is a subcommand or group. Otherwise it's an argument
	optType := opt.Options[0].Type
	if optType != 1 && optType != 2 {
		return opt
	}

	return traverseCommandOption(opt.Options[0])
}

func GetActiveSubCommand(cdata discordgo.ApplicationCommandInteractionData) *discordgo.ApplicationCommandInteractionDataOption {
	if len(cdata.Options) == 0 {
		return nil
	}

	return traverseCommandOption(cdata.Options[0])
}

type FormatFunc[T any, V any] = func(item T, index int) (name string, value V)

// Creates a slice of [discordgo.ApplicationCommandOptionChoice] by transforming the given items slice.
// The Name and Value are acquired from the given format function (where T is the item).
// This resulting slice is then capped at the max Discord allows to prevent the command from failing.
//
// NOTE: Formatting like **bold** or italics won't work in autocomplete so don't use those when formatting Name.
func CreateAutocompleteChoices[T any, V any](items []T, format FormatFunc[T, V]) []*discordgo.ApplicationCommandOptionChoice {
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, len(items))
	for i, item := range items {
		name, value := format(item, i)
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  name,
			Value: value,
		})

		if len(choices) == AUTOCOMPLETE_CHOICE_LIMIT {
			break
		}
	}

	return choices
}
