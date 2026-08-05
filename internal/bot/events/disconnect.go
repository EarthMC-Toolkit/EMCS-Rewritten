package events

import (
	"emcsrw/pkg/utils/logutil"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
)

var ShuttingDown atomic.Bool

func OnDisconnect(s *discordgo.Session, r *discordgo.Disconnect) {
	if ShuttingDown.Load() {
		return
	}

	logutil.Println(logutil.YELLOW, "WARN | Disconnected from Discord gateway. Discordgo should auto reconnect...")
}
