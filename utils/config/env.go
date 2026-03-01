// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Retrieves an OS environment variable by name,
// failing with an error if non-existent or empty.
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

// Parses an environment variable as the desired type,
// failing with an error if not possible.
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
			return any(uint(u)).(T), nil
		case uint8:
			return any(uint8(u)).(T), nil
		case uint16:
			return any(uint16(u)).(T), nil
		case uint32:
			return any(uint32(u)).(T), nil
		case uint64:
			return any(uint64(u)).(T), nil
		}

	// signed ints
	case int, int8, int16, int32, int64:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as int: %v", v, err)
		}
		switch any(zero).(type) {
		case int:
			return any(int(n)).(T), nil
		case int8:
			return any(int8(n)).(T), nil
		case int16:
			return any(int16(n)).(T), nil
		case int32:
			return any(int32(n)).(T), nil
		case int64:
			return any(int64(n)).(T), nil
		}

	// floats
	case float32, float64:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as float: %v", v, err)
		}
		switch any(zero).(type) {
		case float32:
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
		fmt.Printf("\nERROR | Custom API port defaulted to 7777. Reason:\n\t%s\n", reason)
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

func ShouldServeAPI() bool {
	v, err := GetEnviroVar("ENABLE_API")
	if err != nil {
		if strings.Contains(err.Error(), "must be specified") {
			return false // By default, we don't want to serve if var is missing.
		}

		log.Fatal(err)
	}

	// String exists and not empty. Check it is a valid bool value
	parsed, err := ParseEnviroVar[bool](v)
	if err != nil {
		log.Fatal(err)
	}

	return parsed
}
