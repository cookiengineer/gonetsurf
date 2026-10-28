// Package runtime implements process supervision and resource limits for the
// browser's renderer and broker processes.
package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// RestartPolicy controls when a process is restarted.
type RestartPolicy uint8

const (
	// RestartNever never restarts a process.
	RestartNever RestartPolicy = iota
	// RestartOnFailure restarts only when the process exits unsuccessfully.
	RestartOnFailure
	// RestartAlways restarts after every exit.
	RestartAlways
)

// Spec describes a supervised process.
type Spec struct {
	Path        string
	Args        []string
	Env         []string
	Dir         string
	Stdout      io.Writer
	Stderr      io.Writer
	Policy      RestartPolicy
	MaxRestarts int
	Backoff     time.Duration
	StopTimeout time.Duration
}

// Supervisor starts and restarts one process.
type Supervisor struct {
	spec Spec

	mutex    sync.Mutex
	process  *os.Process
	cancel   context.CancelFunc
	done     chan struct{}
	starts   int
	restarts int
}

// New returns a process supervisor.
func New(spec Spec) *Supervisor {
	if spec.MaxRestarts <= 0 {
		spec.MaxRestarts = 3
	}
	if spec.Backoff <= 0 {
		spec.Backoff = 100 * time.Millisecond
	}
	if spec.StopTimeout <= 0 {
		spec.StopTimeout = 2 * time.Second
	}
	if spec.Stdout == nil {
		spec.Stdout = io.Discard
	}
	if spec.Stderr == nil {
		spec.Stderr = io.Discard
	}
	return &Supervisor{spec: spec}
}

// Run starts the process and restarts it according to the policy. It returns
// nil after Stop or when a non-restartable process exits successfully.
func (supervisor *Supervisor) Run(ctx context.Context) error {
	if supervisor.spec.Path == "" {
		return fmt.Errorf("runtime: empty process path")
	}
	runContext, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan struct{})
	supervisor.mutex.Lock()
	supervisor.cancel = cancel
	supervisor.done = done
	supervisor.mutex.Unlock()
	defer close(done)

	backoff := supervisor.spec.Backoff
	for {
		if runContext.Err() != nil {
			return nil
		}
		supervisor.mutex.Lock()
		supervisor.starts++
		supervisor.mutex.Unlock()

		command := exec.CommandContext(runContext, supervisor.spec.Path, supervisor.spec.Args...)
		command.Env = supervisor.spec.Env
		command.Dir = supervisor.spec.Dir
		command.Stdout = supervisor.spec.Stdout
		command.Stderr = supervisor.spec.Stderr
		startErr := command.Start()
		if startErr != nil {
			return startErr
		}
		supervisor.mutex.Lock()
		supervisor.process = command.Process
		supervisor.mutex.Unlock()
		waitErr := command.Wait()
		supervisor.mutex.Lock()
		supervisor.process = nil
		supervisor.mutex.Unlock()

		if runContext.Err() != nil {
			return nil
		}
		shouldRestart := supervisor.spec.Policy == RestartAlways ||
			(supervisor.spec.Policy == RestartOnFailure && waitErr != nil)
		if !shouldRestart {
			if waitErr != nil {
				return waitErr
			}
			return nil
		}
		supervisor.mutex.Lock()
		if supervisor.restarts >= supervisor.spec.MaxRestarts {
			supervisor.mutex.Unlock()
			return fmt.Errorf("runtime: %s exceeded %d restarts", supervisor.spec.Path, supervisor.spec.MaxRestarts)
		}
		supervisor.restarts++
		supervisor.mutex.Unlock()

		timer := time.NewTimer(backoff)
		select {
		case <-runContext.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}
}

// Stop terminates the running process, escalating after StopTimeout.
func (supervisor *Supervisor) Stop() error {
	supervisor.mutex.Lock()
	process := supervisor.process
	cancel := supervisor.cancel
	done := supervisor.done
	supervisor.mutex.Unlock()
	if cancel != nil {
		cancel()
	}
	if process == nil {
		return nil
	}
	_ = process.Signal(syscall.SIGTERM)
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-time.After(supervisor.spec.StopTimeout):
		_ = process.Kill()
		<-done
		return nil
	}
}

// Starts returns the number of process starts.
func (supervisor *Supervisor) Starts() int {
	supervisor.mutex.Lock()
	defer supervisor.mutex.Unlock()
	return supervisor.starts
}

// Restarts returns the number of restarts.
func (supervisor *Supervisor) Restarts() int {
	supervisor.mutex.Lock()
	defer supervisor.mutex.Unlock()
	return supervisor.restarts
}

// PID returns the current process identifier, or zero.
func (supervisor *Supervisor) PID() int {
	supervisor.mutex.Lock()
	defer supervisor.mutex.Unlock()
	if supervisor.process == nil {
		return 0
	}
	return supervisor.process.Pid
}
