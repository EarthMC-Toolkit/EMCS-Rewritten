//go:build linux

package monitor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type process struct {
	pid      int
	command  string
	cpuTicks uint64
	rssBytes uint64
}

func Run() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find monitor executable: %w", err)
	}
	appName := filepath.Base(executable)
	interactive := isTerminal(os.Stdout)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	previousProcesses := make(map[int]process)
	previousSystemTicks, err := readSystemCPUTicks()
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		processes, err := findAppProcesses(appName)
		if err != nil {
			return err
		}
		systemTicks, err := readSystemCPUTicks()
		if err != nil {
			return err
		}
		totalMemory, err := readTotalMemory()
		if err != nil {
			return err
		}

		if interactive {
			fmt.Fprint(os.Stdout, "\033[H\033[2J")
		}
		fmt.Printf("EMCS process monitor  %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf("%-8s %-8s %12s %12s %8s\n", "PID", "COMMAND", "CPU%*", "RSS MiB", "%MEM")

		sort.Slice(processes, func(i, j int) bool {
			if processes[i].command == processes[j].command {
				return processes[i].pid < processes[j].pid
			}
			return processes[i].command < processes[j].command
		})

		if len(processes) == 0 {
			fmt.Println("No bot or API process found. Start them with `go run . bot` and `go run . api`.")
		}

		for _, current := range processes {
			cpu := "n/a"
			if previous, ok := previousProcesses[current.pid]; ok && systemTicks > previousSystemTicks {
				processDelta := current.cpuTicks - previous.cpuTicks
				systemDelta := systemTicks - previousSystemTicks
				// Match top's per-core percentage: one fully used logical CPU is 100%.
				cpu = fmt.Sprintf("%.1f", float64(processDelta)/float64(systemDelta)*float64(runtime.NumCPU())*100)
			}

			rssMiB := float64(current.rssBytes) / (1024 * 1024)
			memPercent := float64(current.rssBytes) / float64(totalMemory) * 100
			fmt.Printf("%-8d %-8s %12s %12.1f %7.2f%%\n", current.pid, current.command, cpu, rssMiB, memPercent)
		}

		fmt.Println("* CPU is per-core (100% means one logical CPU fully used). Ctrl+C exits.")
		previousProcesses = make(map[int]process, len(processes))
		for _, p := range processes {
			previousProcesses[p.pid] = p
		}
		previousSystemTicks = systemTicks

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func findAppProcesses(appName string) ([]process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := make([]process, 0, 2)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}

		cmdline, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "cmdline"))
		if err != nil {
			continue // Process exited or became inaccessible while scanning.
		}
		args := strings.Split(string(cmdline), "\x00")
		if len(args) < 2 || executableName(args[0]) != appName {
			continue
		}

		command := ""
		for _, arg := range args[1:] {
			if arg == "bot" || arg == "api" {
				command = arg
				break
			}
		}
		if command == "" {
			continue
		}

		cpuTicks, err := readProcessCPUTicks(pid)
		if err != nil {
			continue
		}
		rssBytes, err := readProcessRSS(pid)
		if err != nil {
			continue
		}
		processes = append(processes, process{
			pid:      pid,
			command:  command,
			cpuTicks: cpuTicks,
			rssBytes: rssBytes,
		})
	}

	return processes, nil
}

func executableName(path string) string {
	return strings.TrimSuffix(filepath.Base(path), " (deleted)")
}

func readProcessCPUTicks(pid int) (uint64, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return 0, err
	}

	// The comm field is parenthesized and may itself contain spaces.
	closeParen := strings.LastIndexByte(string(data), ')')
	if closeParen < 0 {
		return 0, fmt.Errorf("malformed process stat for PID %d", pid)
	}
	fields := strings.Fields(string(data)[closeParen+1:])
	if len(fields) <= 12 {
		return 0, fmt.Errorf("incomplete process stat for PID %d", pid)
	}
	userTicks, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse user CPU ticks for PID %d: %w", pid, err)
	}
	systemTicks, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse system CPU ticks for PID %d: %w", pid, err)
	}
	return userTicks + systemTicks, nil
}

func readProcessRSS(pid int) (uint64, error) {
	data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "statm"))
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, fmt.Errorf("incomplete process memory stats for PID %d", pid)
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse RSS pages for PID %d: %w", pid, err)
	}
	return pages * uint64(os.Getpagesize()), nil
}

func readSystemCPUTicks() (uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, fmt.Errorf("read system CPU stats: %w", err)
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 9 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid aggregate CPU stats in /proc/stat")
	}

	var ticks uint64
	for _, field := range fields[1:9] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse aggregate CPU stats: %w", err)
		}
		ticks += value
	}
	return ticks, nil
}

func readTotalMemory() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, fmt.Errorf("read system memory stats: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kib, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse total memory: %w", err)
		}
		return kib * 1024, nil
	}
	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
