package events

import (
	"emcsrw/pkg/utils/logutil"
	"sync/atomic"

	"github.com/bwmarrin/discordgo"
)

var ShuttingDown atomic.Bool

func OnConnect(s *discordgo.Session, r *discordgo.Connect) {
	logutil.FileLog.Printf("EMSCRW | Gateway connected | SessionID=%s\n", s.State.SessionID)
}

func OnDisconnect(s *discordgo.Session, r *discordgo.Disconnect) {
	if !ShuttingDown.Load() {
		logutil.Println(logutil.YELLOW, "WARN | Discord gateway: disconnect event fired. DiscordGo should auto reconnect..")
	}

	logutil.FileLog.Printf("EMSCRW | Gateway disconnected | SessionID=%s\n", s.State.SessionID)
}

func OnResumed(s *discordgo.Session, event *discordgo.Resumed) {
	if !ShuttingDown.Load() {
		logutil.Println(logutil.GREEN, "INFO | Discord gateway: resumed event fired.")
	}

	logutil.FileLog.Printf("EMSCRW | Gateway resumed | SessionID=%s | Trace=%v\n", s.State.SessionID, event.Trace)
}
