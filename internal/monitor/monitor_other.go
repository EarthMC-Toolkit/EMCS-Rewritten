//go:build !linux

package monitor

import "fmt"

func Run() error {
	return fmt.Errorf("the process monitor is currently supported on Linux only")
}
