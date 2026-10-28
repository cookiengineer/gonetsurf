package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRestartOnFailure(t *testing.T) {
	supervisor := New(Spec{
		Path:        "/bin/sh",
		Args:        []string{"-c", "exit 1"},
		Policy:      RestartOnFailure,
		MaxRestarts: 2,
		Backoff:     time.Millisecond,
	})
	err := supervisor.Run(context.Background())
	if err == nil {
		t.Fatal("expected a failure after the restart budget")
	}
	if supervisor.Starts() != 3 || supervisor.Restarts() != 2 {
		t.Errorf("starts = %d restarts = %d", supervisor.Starts(), supervisor.Restarts())
	}
}

func TestNoRestartOnSuccess(t *testing.T) {
	supervisor := New(Spec{
		Path:   "/bin/sh",
		Args:   []string{"-c", "exit 0"},
		Policy: RestartOnFailure,
	})
	if err := supervisor.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if supervisor.Starts() != 1 || supervisor.Restarts() != 0 {
		t.Errorf("starts = %d restarts = %d", supervisor.Starts(), supervisor.Restarts())
	}
}

func TestStopRunningProcess(t *testing.T) {
	supervisor := New(Spec{
		Path:        "/bin/sh",
		Args:        []string{"-c", "sleep 10"},
		Policy:      RestartAlways,
		StopTimeout: time.Second,
	})
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(context.Background()) }()
	deadline := time.Now().Add(2 * time.Second)
	for supervisor.PID() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if supervisor.PID() == 0 {
		t.Fatal("process did not start")
	}
	if err := supervisor.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop")
	}
	if supervisor.Starts() != 1 {
		t.Errorf("starts = %d, want 1", supervisor.Starts())
	}
}

func TestMissingCommand(t *testing.T) {
	supervisor := New(Spec{Path: "/definitely/not/a/command"})
	if err := supervisor.Run(context.Background()); err == nil {
		t.Fatal("expected a start error")
	}
}

func TestContextCancellation(t *testing.T) {
	supervisor := New(Spec{Path: "/bin/sh", Args: []string{"-c", "sleep 10"}, Policy: RestartAlways})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- supervisor.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Run = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancellation")
	}
}
