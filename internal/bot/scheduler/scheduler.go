package scheduler

import (
	"emcsrw/pkg/utils/logutil"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Scheduler struct {
	wg       sync.WaitGroup
	stopping atomic.Bool // Bot shutdown hint. Simple state used to allow or prevent new tasks.
	//tasks map[string]func() // Task name -> task func.
	//doneCh chan string      // Channel for logging task completions.
}

var Instance *Scheduler

func New() *Scheduler {
	// Commented for now until we require actual use of a Context.
	//
	// Right now, an atomic 'stopping' bool is enough as it prevents new tasks
	// from running and allows current ones to complete before shutdown.
	//
	// A context.Context would only become useful when we need to cancel work
	// that's already running by propagating the context, for example with
	// http.NewRequestWithContext() or exec.CommandContext().
	//
	// ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		//tasks: make(map[string]func()),
		//doneCh: make(chan string, 32),
		//ctx: ctx
		//cancel: cancel
	}
}

func (s *Scheduler) Schedule(taskName string, task func(), runInitial bool, interval time.Duration) {
	s.wg.Go(func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		if runInitial && !s.stopping.Load() {
			task()
		}

		for range ticker.C {
			if s.stopping.Load() {
				return // hint was given to stop. prevent new ticks and therefore new tasks from starting.
			}

			task()
			if !s.stopping.Load() {
				continue // task finished before shutdown started
			}

			fmt.Println()
			logutil.Logf(logutil.BLUE, "[Scheduler]: Task '%s' finished during shutdown.\n", taskName)
		}
	})
}

// Shutdown stops this scheduler from running new tasks and waits up to timeoutDuration for all tasks to finish.
// Returns a status string indicating success or timeout.
func (s *Scheduler) Shutdown(timeoutDuration time.Duration) string {
	s.stopping.Store(true) // hint to scheduler to prevent scheduling future tasks.

	done := make(chan struct{})
	go func() {
		s.wg.Wait() // wait for scheduler goroutines and any tasks they are running
		close(done)
	}()

	select {
	case <-done:
		return "All tasks finished"
	case <-time.After(timeoutDuration):
		return "Timeout reached, exiting.."
	}
}
