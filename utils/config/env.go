// TODO: Extend this file to support actual config system instead of only .env
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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
