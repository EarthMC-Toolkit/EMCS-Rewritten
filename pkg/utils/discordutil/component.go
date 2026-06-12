package discordutil

import "github.com/bwmarrin/discordgo"

func Choice[T any](name string, value T) *discordgo.ApplicationCommandOptionChoice {
	return &discordgo.ApplicationCommandOptionChoice{
		Name:  name,
		Value: value,
	}
}

func CommandOption(
	optType discordgo.ApplicationCommandOptionType,
	name, description string,
	options ...*discordgo.ApplicationCommandOption,
) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        optType,
		Name:        name,
		Description: description,
		Options:     options,
	}
}

func SubcommandOption(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return CommandOption(discordgo.ApplicationCommandOptionSubCommand, name, description, options...)
}

func SubcommandGroupOption(name, description string, options ...*discordgo.ApplicationCommandOption) *discordgo.ApplicationCommandOption {
	return CommandOption(discordgo.ApplicationCommandOptionSubCommandGroup, name, description, options...)
}

func BoolOption(name, description string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        name,
		Description: description,
	}
}

func StringOption(
	name, description string,
	minLen *int, maxLen *int,
	choices ...*discordgo.ApplicationCommandOptionChoice,
) *discordgo.ApplicationCommandOption {
	max := 0
	if maxLen != nil {
		max = *maxLen
	}

	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        name,
		Description: description,
		MinLength:   minLen,
		MaxLength:   max,
		Choices:     choices,
	}
}

func RequiredStringOption(name, description string, minLen, maxLen int) *discordgo.ApplicationCommandOption {
	opt := StringOption(name, description, &minLen, &maxLen)
	opt.Required = true
	return opt
}

func AutocompleteStringOption(name, description string, minLen, maxLen int, required bool) *discordgo.ApplicationCommandOption {
	opt := StringOption(name, description, &minLen, &maxLen)
	opt.Required = required
	opt.Autocomplete = true
	return opt
}

func IntegerOption(name, description string, minVal, maxVal int, required bool) *discordgo.ApplicationCommandOption {
	min := float64(minVal)
	max := float64(maxVal)

	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        name,
		Description: description,
		MinValue:    &min,
		MaxValue:    max,
		Required:    required,
	}
}

func RequiredNumberOption(name, description string, minVal, maxVal float64) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionNumber,
		Name:        name,
		Description: description,
		MinValue:    &minVal,
		MaxValue:    maxVal,
		Required:    true,
	}
}

func TextInputActionRow(input discordgo.TextInput) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{input},
	}
}

func TextInput(style discordgo.TextInputStyle, cid, label, placeholder string, minLen uint, maxLen uint) discordgo.TextInput {
	return discordgo.TextInput{
		CustomID:    cid,
		Label:       label,
		Placeholder: placeholder,
		MinLength:   int(minLen),
		MaxLength:   int(maxLen),
		Style:       style,
	}
}

func TextInputShort(cid, label, placeholder string, minLen uint, maxLen uint) discordgo.TextInput {
	return TextInput(discordgo.TextInputShort, cid, label, placeholder, minLen, maxLen)
}

func TextInputParagraph(cid, label, placeholder string, minLen uint, maxLen uint) discordgo.TextInput {
	return TextInput(discordgo.TextInputParagraph, cid, label, placeholder, minLen, maxLen)
}

func RequiredTextInputShort(cid, label, placeholder string, minLen uint, maxLen uint) discordgo.TextInput {
	ti := TextInput(discordgo.TextInputShort, cid, label, placeholder, minLen, maxLen)
	ti.Required = true
	return ti
}

func RequiredTextInputParagraph(cid, label, placeholder string, minLen uint, maxLen uint) discordgo.TextInput {
	ti := TextInput(discordgo.TextInputParagraph, cid, label, placeholder, minLen, maxLen)
	ti.Required = true
	return ti
}

func SelectMenuActionRow(menu discordgo.SelectMenu) discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{menu},
	}
}

func ButtonActionRow(btns ...discordgo.Button) discordgo.ActionsRow {
	components := []discordgo.MessageComponent{}
	for _, btn := range btns {
		components = append(components, btn)
	}

	return discordgo.ActionsRow{Components: components}
}

func SelectMenuOption(label, value, desc string, isDefault bool) discordgo.SelectMenuOption {
	return discordgo.SelectMenuOption{
		Label:       label,
		Value:       value,
		Description: desc,
		Default:     isDefault,
	}
}

func SelectMenuOptionEmoji(label, value, desc string, isDefault bool, emoji *discordgo.ComponentEmoji) discordgo.SelectMenuOption {
	return discordgo.SelectMenuOption{
		Label:       label,
		Value:       value,
		Description: desc,
		Emoji:       emoji,
		Default:     false,
	}
}

// Flags: IsMessageComponentsV2 must also be set when using this new label component.
func Label(label string, description string, component discordgo.MessageComponent) discordgo.Label {
	return discordgo.Label{
		Label:       label,
		Description: description,
		Component:   component,
	}
}
