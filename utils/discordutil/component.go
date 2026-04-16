package discordutil

import "github.com/bwmarrin/discordgo"

func BoolOption(name, description string) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionBoolean,
		Name:        name,
		Description: description,
	}
}

func StringOption(name, description string, minLen *int, maxLen int) *discordgo.ApplicationCommandOption {
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionString,
		Name:        name,
		Description: description,
		MinLength:   minLen,
		MaxLength:   maxLen,
	}
}

func RequiredStringOption(name, description string, minLen, maxLen int) *discordgo.ApplicationCommandOption {
	opt := StringOption(name, description, &minLen, maxLen)
	opt.Required = true
	return opt
}

func AutocompleteStringOption(name, description string, minLen, maxLen int, required bool) *discordgo.ApplicationCommandOption {
	opt := StringOption(name, description, &minLen, maxLen)
	opt.Required = required
	opt.Autocomplete = true
	return opt
}

func RequiredIntegerOption(name, description string, minVal, maxVal int) *discordgo.ApplicationCommandOption {
	min, max := float64(minVal), float64(maxVal)
	return &discordgo.ApplicationCommandOption{
		Type:        discordgo.ApplicationCommandOptionInteger,
		Name:        name,
		Description: description,
		MinValue:    &min,
		MaxValue:    max,
		Required:    true,
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
