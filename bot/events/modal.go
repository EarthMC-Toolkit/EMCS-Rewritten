package events

import (
	"emcsrw/bot/slashcommands"
	"emcsrw/utils/discordutil"
	"log"
	"runtime/debug"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func OnModalSubmitInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("handler OnModalSubmitInteractionCreate recovered from a panic.\n%v\n%s", err, debug.Stack())
			discordutil.ReplyWithPanicError(s, i.Interaction, err)
		}
	}()

	if i.Type != discordgo.InteractionModalSubmit {
		return
	}

	data := i.ModalSubmitData()
	cmdName, _, _ := strings.Cut(data.CustomID, "_")
	if cmd, ok := slashcommands.All()[cmdName]; ok {
		if modalCmd, ok := cmd.(slashcommands.ModalHandler); ok {
			_ = modalCmd.HandleModal(s, i.Interaction, data.CustomID)
		}
	}
}
