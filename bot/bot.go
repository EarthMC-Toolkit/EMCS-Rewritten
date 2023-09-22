package bot

import (
	"fmt"
)

var (
	BotToken string
)

func Run() {
	fmt.Println("Found token: ", BotToken)
}