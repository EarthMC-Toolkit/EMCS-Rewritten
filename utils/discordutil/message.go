package discordutil

import "github.com/bwmarrin/discordgo"

var WIKI_EMOJI = discordgo.ComponentEmoji{Name: "📰"}
var DISCORD_EMOJI = discordgo.ComponentEmoji{
	Name: "discordlogo",
	ID:   "1513955352608243923",
}

// Data shared between both an interaction and a message. This means no matter the method we choose to
// use to output data from the builder, these fields will be available on the result.
type SharedOpts struct {
	Content         string                            `json:"content,omitempty"`
	Embeds          []*discordgo.MessageEmbed         `json:"embeds"`
	Components      []discordgo.MessageComponent      `json:"components"`
	Files           []*discordgo.File                 `json:"-"`
	TTS             bool                              `json:"tts"`
	AllowedMentions *discordgo.MessageAllowedMentions `json:"allowed_mentions,omitempty"`
	Poll            *discordgo.Poll                   `json:"poll,omitempty"`
	Flags           discordgo.MessageFlags            `json:"flags,omitempty"`
}

// Options specific to [discordgo.MessageSend] only.
type MessageOpts struct {
	Reference  *discordgo.MessageReference `json:"message_reference,omitempty"`
	StickerIDs []string                    `json:"sticker_ids"`
}

// Options specific to [discordgo.InteractionResponseData] only.
type InteractionOpts struct {
	Choices  []*discordgo.ApplicationCommandOptionChoice `json:"choices,omitempty"`
	CustomID string                                      `json:"custom_id,omitempty"`
	Title    string                                      `json:"title,omitempty"`
}

// Options specific to [discordgo.WebhookParams] only.
type WebhookOpts struct {
	Username   string `json:"username,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	ThreadName string `json:"thread_name,omitempty"`
}

type DispatchType int

const (
	DispatchTypeMessage     DispatchType = iota // Uses MessageSend
	DispatchTypeInteraction                     // Uses InteractionResponseData
	DispatchTypeFollowup                        // Uses WebhookParams
)

type MessageBuilder struct {
	SharedOpts
	msgOpts         *MessageOpts
	interactionOpts *InteractionOpts
	webhookOpts     *WebhookOpts

	Rows []discordgo.ActionsRow
}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

func (b *MessageBuilder) SetContent(content *string) {
	var c string
	if content != nil {
		c = *content
	}

	b.Content = c
}

func (b *MessageBuilder) AddEmbed(embed *discordgo.MessageEmbed) *MessageBuilder {
	b.Embeds = append(b.Embeds, embed)
	return b
}

// NOTE: Only a button with the LinkButton style can have a link. Also, URL is mutually exclusive with CustomID.
func (b *MessageBuilder) AddButton(
	label string, style discordgo.ButtonStyle,
	url *string, emoji *discordgo.ComponentEmoji,
	customID *string,
) *MessageBuilder {
	btn := discordgo.Button{
		Label: label,
		Style: style,
		Emoji: emoji,
	}

	// enforce Discord rule: link vs interaction button
	if style == discordgo.LinkButton {
		if url == nil {
			panic("link button requires URL")
		}
		btn.URL = *url
	} else {
		if customID == nil {
			panic("non-link button requires customID")
		}
		btn.CustomID = *customID
	}

	b.AppendComponent(btn)
	return b
}

func (b *MessageBuilder) AppendComponent(btn discordgo.MessageComponent) *MessageBuilder {
	if len(b.Rows) == 0 {
		b.Rows = append(b.Rows, discordgo.ActionsRow{})
	}

	lastIdx := len(b.Rows) - 1
	if len(b.Rows[lastIdx].Components) >= 5 {
		b.Rows = append(b.Rows, discordgo.ActionsRow{})
		lastIdx++
	}

	b.Rows[lastIdx].Components = append(b.Rows[lastIdx].Components, btn)
	return b
}

func (b *MessageBuilder) BuildComponents() []discordgo.MessageComponent {
	components := make([]discordgo.MessageComponent, len(b.Rows))
	for i := range b.Rows {
		components[i] = b.Rows[i]
	}

	return components
}

func (b *MessageBuilder) MessageData() *discordgo.MessageSend {
	m := &discordgo.MessageSend{
		Content:         b.Content,
		TTS:             b.TTS,
		Embeds:          b.Embeds,
		Components:      b.BuildComponents(),
		AllowedMentions: b.AllowedMentions,
		Files:           b.Files,
		Flags:           b.Flags,
		Poll:            b.Poll,
	}
	if b.msgOpts != nil {
		m.Reference = b.msgOpts.Reference
		m.StickerIDs = b.msgOpts.StickerIDs
	}

	return m
}

func (b *MessageBuilder) InteractionData() *discordgo.InteractionResponseData {
	i := &discordgo.InteractionResponseData{
		Content:         b.Content,
		TTS:             b.TTS,
		Embeds:          b.Embeds,
		Components:      b.BuildComponents(),
		AllowedMentions: b.AllowedMentions,
		Files:           b.Files,
		Flags:           b.Flags,
		Poll:            b.Poll,
	}
	if b.interactionOpts != nil {
		i.Choices = b.interactionOpts.Choices
		i.CustomID = b.interactionOpts.CustomID
		i.Title = b.interactionOpts.Title
	}

	return i
}

func (b *MessageBuilder) WebhookData() *discordgo.WebhookParams {
	w := &discordgo.WebhookParams{
		Content:         b.Content,
		TTS:             b.TTS,
		Embeds:          b.Embeds,
		Components:      b.BuildComponents(),
		AllowedMentions: b.AllowedMentions,
		Files:           b.Files,
		Flags:           b.Flags,
	}
	if b.webhookOpts != nil {
		w.Username = b.webhookOpts.Username
		w.AvatarURL = b.webhookOpts.AvatarURL
		w.ThreadName = b.webhookOpts.ThreadName
	}

	return w
}

// func (b *MessageBuilder) Send(s *discordgo.Session, channelID string, i ...*discordgo.Interaction) (*discordgo.Message, error) {
// 	switch b.target {
// 	case DispatchTypeMessage:
// 		return s.ChannelMessageSendComplex(channelID, b.MessageData())
// 	case DispatchTypeInteraction:
// 		return nil, s.InteractionRespond(i[0], &discordgo.InteractionResponse{
// 			Type: discordgo.InteractionResponseChannelMessageWithSource,
// 			Data: b.InteractionData(),
// 		})
// 	case DispatchTypeFollowup:
// 		return s.FollowupMessageCreate(i[0], true, b.WebhookData())
// 	default:
// 		return nil, nil
// 	}
// }
