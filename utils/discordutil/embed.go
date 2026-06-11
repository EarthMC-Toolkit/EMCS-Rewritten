package discordutil

import (
	"github.com/bwmarrin/discordgo"
)

const EMBED_FIELD_VALUE_LIMIT = 1024
const EMBED_DESCRIPTION_LIMIT = 4096

var (
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

var DEFAULT_FOOTER = &discordgo.MessageEmbedFooter{
	IconURL: "https://cdn.discordapp.com/avatars/263377802647175170/a_0cd469f208f88cf98941123eb1b52259.webp?size=512&animated=true",
	Text:    "Maintained by Owen3H • Open Source on GitHub 💛", // unless you maintain your own fork, pls keep this as is :)
}

// TODO: Maybe create a CustomEmbed that wraps MessageEmbed and adds these methods?
// #region Embed field helper funcs

func NewEmbedFieldSpacer(inline bool) *discordgo.MessageEmbedField {
	return NewEmbedField("", "", inline)
}

func NewEmbedField(name string, value string, inline bool) *discordgo.MessageEmbedField {
	return &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	}
}

func AddField(embed *discordgo.MessageEmbed, name string, value string, inline bool) {
	embed.Fields = append(embed.Fields, NewEmbedField(name, value, inline))
}

func PrependField(embed *discordgo.MessageEmbed, name string, value string, inline bool) {
	embed.Fields = append([]*discordgo.MessageEmbedField{NewEmbedField(name, value, inline)}, embed.Fields...)
}

//#endregion

// Helper for building a Discord embed in a more convenient and programmatic way.
type EmbedBuilder discordgo.MessageEmbed

func NewEmbedBuilder(colour *int, title *string, description *string, footer *discordgo.MessageEmbedFooter) *EmbedBuilder {
	e := &EmbedBuilder{Type: discordgo.EmbedTypeRich, Color: DEFAULT, Footer: DEFAULT_FOOTER}
	if colour != nil {
		e.Color = *colour
	}
	if title != nil {
		e.Title = *title
	}
	if description != nil {
		e.Description = *description
	}
	if footer != nil {
		e.Footer = footer
	}

	return e
}

func (e *EmbedBuilder) SetType(t discordgo.EmbedType) *EmbedBuilder {
	e.Type = t
	return e
}

func (e *EmbedBuilder) SetURL(url string) *EmbedBuilder {
	e.URL = url
	return e
}

func (e *EmbedBuilder) SetTitle(title string) *EmbedBuilder {
	e.Title = title
	return e
}

func (e *EmbedBuilder) SetDescription(description string) *EmbedBuilder {
	e.Description = description
	return e
}

func (e *EmbedBuilder) SetColour(color int) *EmbedBuilder {
	e.Color = color
	return e
}

func (e *EmbedBuilder) SetFooter(text string, iconURL *string) *EmbedBuilder {
	e.Footer = &discordgo.MessageEmbedFooter{Text: text}
	if iconURL != nil {
		e.Footer.IconURL = *iconURL
	}

	return e
}

func (e *EmbedBuilder) SetThumbnail(url string, proxyUrl *string) *EmbedBuilder {
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	if proxyUrl != nil {
		e.Thumbnail.ProxyURL = *proxyUrl
	}

	return e
}

func (e *EmbedBuilder) SetThumbnailSize(width, height int) *EmbedBuilder {
	e.Thumbnail.Width = width
	e.Thumbnail.Height = height
	return e
}

func (e *EmbedBuilder) SetImage(url string, proxyUrl *string) *EmbedBuilder {
	e.Image = &discordgo.MessageEmbedImage{URL: url}
	if proxyUrl != nil {
		e.Image.ProxyURL = *proxyUrl
	}

	return e
}

func (e *EmbedBuilder) SetImageSize(width, height int) *EmbedBuilder {
	e.Image.Width = width
	e.Image.Height = height
	return e
}

func (e *EmbedBuilder) SetAuthor(name string, url *string, iconURL *string) *EmbedBuilder {
	e.Author = &discordgo.MessageEmbedAuthor{Name: name}
	if url != nil {
		e.Author.URL = *url
	}
	if iconURL != nil {
		e.Author.IconURL = *iconURL
	}

	return e
}

func (e *EmbedBuilder) SetTimestamp(timestamp string) *EmbedBuilder {
	e.Timestamp = timestamp
	return e
}

func (b *EmbedBuilder) SetFields(fields ...*discordgo.MessageEmbedField) {
	(*discordgo.MessageEmbed)(b).Fields = fields // write to underlying embed instead of the builder
}

func (e *EmbedBuilder) AddField(name string, value string, inline bool) *EmbedBuilder {
	e.Fields = append(e.Fields, NewEmbedField(name, value, inline))
	return e
}

func (e *EmbedBuilder) PrependField(name string, value string, inline bool) *EmbedBuilder {
	e.Fields = append([]*discordgo.MessageEmbedField{NewEmbedField(name, value, inline)}, e.Fields...)
	return e
}

func (b *EmbedBuilder) Build() *discordgo.MessageEmbed {
	e := discordgo.MessageEmbed(*b) // cast back to original embed
	return &e
}
