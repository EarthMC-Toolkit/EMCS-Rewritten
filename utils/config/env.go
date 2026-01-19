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
		return "", fmt.Errorf("environment var %q must be specified", name)
	}
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("environment var %q must not be empty", name)
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
		b, err := strconv.ParseBool(v)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as bool: %v", v, err)
		}
		return any(b).(T), nil

	// signed ints
	case int, int32, int64:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as int: %v", v, err)
		}
		switch any(zero).(type) {
		case int:
			return any(int(n)).(T), nil
		case int32:
			return any(int32(n)).(T), nil
		case int64:
			return any(int64(n)).(T), nil
		}

	// unsigned ints
	case uint, uint32, uint64:
		u, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse environment var %q as uint: %v", v, err)
		}
		switch any(zero).(type) {
		case uint:
			return any(uint(u)).(T), nil
		case uint32:
			return any(uint32(u)).(T), nil
		case uint64:
			return any(uint64(u)).(T), nil
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
