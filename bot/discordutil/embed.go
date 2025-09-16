package discordutil

import (
	"github.com/bwmarrin/discordgo"
)

const (
	DEFAULT     = 0x000000
	WHITE       = 0xffffff
	AQUA        = 0x1abc9c
	GREEN       = 0x2ecc71
	BLUE        = 0x3498db
	YELLOW      = 0xffff00
	GOLD        = 0xf1c40f
	PURPLE      = 0x9b59b6
	ORANGE      = 0xe67e22
	RED         = 0xe74c3c
	GREY        = 0x95a5a6
	DARK_AQUA   = 0x11806a
	DARK_GREEN  = 0x1f8b4c
	DARK_BLUE   = 0x206694
	DARK_PURPLE = 0x71368a
	DARK_GOLD   = 0xc27c0e
	DARK_ORANGE = 0xa84300
	DARK_RED    = 0x992d22
	DARK_GREY   = 0x979c9f
	BLURPLE     = 0x7289da
	DARK        = 0x2c2f33
)

type LabelledValue struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func EmbedField(name string, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

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
