// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnv(prod bool) error {
	files := []string{".env.local", ".env.dev"}
	if prod {
		files = []string{".env", ".env.prod", ".env.production"}
	}
	return godotenv.Load(files...)
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
