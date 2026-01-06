// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func GetEnviroVar(name string) (string, error) {
	v, found := os.LookupEnv(name)
	if !found {
		return "", fmt.Errorf("environment variable %q must be specified", name)
	}
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("environment variable %q must not be empty", name)
	}

	return v, nil
}

// Parses an EnviroVar to the desired type
func ParseEnviroVar[T any](v string) (T, error) {
	var zero T

	switch any(zero).(type) {
	case string:
		return any(v).(T), nil
	case bool:
		val, err := strconv.ParseBool(v)
		if err != nil {
			return zero, fmt.Errorf("failed to parse %q as bool: %v", v, err)
		}

		return any(val).(T), nil
	case int:
		val, err := strconv.Atoi(v)
		if err != nil {
			return zero, fmt.Errorf("failed to parse %q as int: %v", v, err)
		}

		return any(val).(T), nil
	case int64:
		val, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse %q as int64: %v", v, err)
		}

		return any(val).(T), nil
	case float64:
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse %q as float64: %v", v, err)
		}

		return any(val).(T), nil
	}

	return zero, fmt.Errorf("unsupported environment variable type %T", zero)
}
