// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

// Owen3H pfp
// var icon = "https://cdn.discordapp.com/avatars/263377802647175170/a_0cd469f208f88cf98941123eb1b52259.webp?size=512&animated=true"

// TODO: Migrate this to .env file, config.json or similar. This is a temporary solution for now.
func GetFooter() *discordgo.MessageEmbedFooter {
	return &discordgo.MessageEmbedFooter{
		IconURL: "https://cdn.discordapp.com/attachments/974491955864150046/1548933270098415667/image.png",
		Text:    "EMCS is open source on GitHub. PRs welcome! 💛", // unless you maintain your own fork, pls keep this as is :)
	}
}

// Retrieves an OS environment variable by name, failing with an error if non-existent or empty.
func GetEnviroVar(name string) (string, error) {
	v, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment var %q must be specified", name)
	}
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("environment var %q must not be empty", name)
	}

	return v, nil
}

// Parses an environment variable as the desired type, failing with an error if not possible.
func ParseEnviroVar[T any](v string) (T, error) {
	var zero T

	switch any(zero).(type) {
	case string:
		return any(v).(T), nil
	case bool:
	case uint, uint8, uint16, uint32, uint64:
	case int, int8, int16, int32, int64:
	case float32, float64:
	default:
		return zero, fmt.Errorf("unsupported environment variable type %T", zero)
	}

	if _, err := fmt.Sscan(v, &zero); err != nil {
		return zero, fmt.Errorf("failed to parse environment var %q as %T: %v", v, zero, err)
	}

	return zero, nil
}

func LoadEnv() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	logutil.DebugLogEnabled, _ = ParseEnviroVar[bool]("ENABLE_DEBUG_LOG")
}

func GetBotToken() string {
	v, err := GetEnviroVar("BOT_TOKEN")
	if err != nil {
		log.Fatal(err)
	}

	// Don't rly need to parse since we already have string
	return v
}

func GetBotID() string {
	v, err := GetEnviroVar("BOT_APP_ID")
	if err != nil {
		log.Fatal(err)
	}

	return v
}

func GetApiPort() uint {
	fail := func(reason string) uint {
		logutil.Logf(logutil.YELLOW, "\nWARN | Custom API port defaulted to 7777. Reason:\n\t%s\n", reason)
		return 7777
	}

	v, err := GetEnviroVar("API_PORT")
	if err != nil {
		return fail(err.Error())
	}

	port, err := ParseEnviroVar[uint](v)
	if err != nil {
		return fail(err.Error())
	}

	switch port {
	case 80, 443:
		return port // Allow HTTP and HTTPS default ports
	default:
		if port < 1024 || port > 49150 {
			return fail("environment variable API_PORT must be 80, 443 or in range 1024-49150")
		}
	}

	return port
}
