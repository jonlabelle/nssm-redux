//go:build windows

package scm

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
)

const rotateControl = svc.Cmd(128)

func Pause(name string) error {
	return control(name, svc.Pause, svc.Paused)
}

func Continue(name string) error {
	return control(name, svc.Continue, svc.Running, svc.StartPending, svc.ContinuePending, svc.Paused)
}

func Rotate(name string) error {
	_, serviceHandle, _, err := openService(name)
	if err != nil {
		return err
	}
	defer func() { _ = serviceHandle.Close() }()

	if _, err := serviceHandle.Control(rotateControl); err != nil {
		return fmt.Errorf("rotate service: %w", err)
	}
	return nil
}

func control(name string, command svc.Cmd, okStates ...svc.State) error {
	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return err
	}
	defer func() { _ = manager.Disconnect() }()
	defer func() { _ = serviceHandle.Close() }()

	if _, err := serviceHandle.Control(command); err != nil {
		return fmt.Errorf("control service: %w", err)
	}

	if len(okStates) == 0 {
		return nil
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		status, err := serviceHandle.Query()
		if err != nil {
			return fmt.Errorf("query service: %w", err)
		}
		for _, state := range okStates {
			if status.State == state {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}

	return fmt.Errorf("timed out waiting for control %d", uint32(command))
}
