package scheduler

import (
	"emcsrw/utils"
	"fmt"
	"sync"
	"time"

	colour "github.com/fatih/color"
)

var (
	HIDDEN = colour.New(colour.FgWhite, colour.Concealed)
)

type Scheduler struct {
	wg       sync.WaitGroup
	tasks    map[string]func() // task name -> task func
	doneCh   chan string       // channel for logging task completions
	stopping bool
}

var Instance *Scheduler

func New() *Scheduler {
	//ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		wg:     sync.WaitGroup{},
		tasks:  make(map[string]func()),
		doneCh: make(chan string, 32),
	}
}

func (s *Scheduler) Schedule(taskName string, task func(), runInitial bool, interval time.Duration) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		if runInitial && !s.stopping {
			task()
		}

		for range ticker.C {
			if s.stopping {
				return // prevent new ticks
			}

			task()
			if s.stopping {
				fmt.Println()
				utils.Logf(HIDDEN, "DEBUG | [Scheduler]: Task '%s' finished during shutdown.\n", taskName)
			}
		}
	}()
}

// Shutdown stops this scheduler from running new tasks and waits up to timeoutDuration for all tasks to finish.
// Returns a status string indicating success or timeout.
func (s *Scheduler) Shutdown(timeoutDuration time.Duration) string {
	s.stopping = true // prevent new ticks

	done := make(chan struct{})
	go func() {
		s.wg.Wait() // wait for currently running tasks
		close(done)
	}()

	select {
	case <-done:
		return "All tasks finished"
	case <-time.After(timeoutDuration):
		return "Timeout reached; exiting"
	}
}
