// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
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
		b, err := strconv.ParseBool(v)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as bool: %v", v, err)
		}
		return any(b).(T), nil

	// unsigned ints
	case uint, uint8, uint16, uint32, uint64:
		u, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as uint: %v", v, err)
		}
		switch any(zero).(type) {
		case uint:
			if strconv.IntSize == 32 && u > math.MaxUint32 {
				return zero, fmt.Errorf("environment var %q exceeds uint range", v)
			}
			return any(uint(u)).(T), nil
		case uint8:
			if u > math.MaxUint8 {
				return zero, fmt.Errorf("environment var %q exceeds uint8 range", v)
			}
			return any(uint8(u)).(T), nil
		case uint16:
			if u > math.MaxUint16 {
				return zero, fmt.Errorf("environment var %q exceeds uint16 range", v)
			}
			return any(uint16(u)).(T), nil
		case uint32:
			if u > math.MaxUint32 {
				return zero, fmt.Errorf("environment var %q exceeds uint32 range", v)
			}
			return any(uint32(u)).(T), nil
		case uint64:
			return any(u).(T), nil
		}

	// signed ints
	case int, int8, int16, int32, int64:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as int: %v", v, err)
		}
		switch any(zero).(type) {
		case int:
			if strconv.IntSize == 32 && (n < math.MinInt32 || n > math.MaxInt32) {
				return zero, fmt.Errorf("environment var %q exceeds int range", v)
			}
			return any(int(n)).(T), nil
		case int8:
			if n < math.MinInt8 || n > math.MaxInt8 {
				return zero, fmt.Errorf("environment var %q exceeds int8 range", v)
			}
			return any(int8(n)).(T), nil
		case int16:
			if n < math.MinInt16 || n > math.MaxInt16 {
				return zero, fmt.Errorf("environment var %q exceeds int16 range", v)
			}
			return any(int16(n)).(T), nil
		case int32:
			if n < math.MinInt32 || n > math.MaxInt32 {
				return zero, fmt.Errorf("environment var %q exceeds int32 range", v)
			}
			return any(int32(n)).(T), nil
		case int64:
			return any(n).(T), nil
		}

	// floats
	case float32, float64:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as float: %v", v, err)
		}
		switch any(zero).(type) {
		case float32:
			if math.IsInf(f, 0) || math.Abs(f) > math.MaxFloat32 {
				return zero, fmt.Errorf("environment var %q exceeds float32 range", v)
			}
			return any(float32(f)).(T), nil
		case float64:
			return any(float64(f)).(T), nil
		}
	}

	return zero, fmt.Errorf("unsupported environment variable type %T", zero)
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
