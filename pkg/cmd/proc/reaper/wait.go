// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package reaper

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// WaitWrapper emulates os/exec.Command.Wait() when reaper is running.
//
// WaitWrapper(true, cmd) should be equivalent to cmd.Wait().
func WaitWrapper(usingReaper bool, notifyCh <-chan ProcessInfo, cmd *exec.Cmd) error {
	if !usingReaper {
		return cmd.Wait()
	}

	var info ProcessInfo

	for info = range notifyCh {
		if info.Pid == cmd.Process.Pid && (info.Status.Exited() || info.Status.Signaled()) {
			break
		}
	}

	err := convertWaitStatus(info.Status)

	// still do cmd.Wait() to release any resources
	waitErr := cmd.Wait()
	if err == nil && waitErr != nil && waitErr.Error() != "waitid: no child processes" {
		err = waitErr
	}

	return err
}

// ProcessWaitWrapper emulates os/exec.Process.Wait() when reaper is running.
// It is equivalent to WaitWrapper
//
// ProcessWaitWrapper(true, proc) should be equivalent to proc.Wait().
func ProcessWaitWrapper(usingReaper bool, notifyCh <-chan ProcessInfo, proc *os.Process) error {
	if !usingReaper {
		_, waitErr := proc.Wait()

		return waitErr
	}

	var info ProcessInfo

	for info = range notifyCh {
		if info.Pid == proc.Pid && (info.Status.Exited() || info.Status.Signaled()) {
			break
		}
	}

	err := convertWaitStatus(info.Status)

	// release any resources
	waitErr := proc.Release()
	if err == nil && waitErr != nil && waitErr.Error() != "waitid: no child processes" {
		err = waitErr
	}

	return err
}

// ExitError is raised when exit status is not equal to 0.
type ExitError struct {
	ExitCode int
}

// Error implements error interface.
func (exitError *ExitError) Error() string {
	return fmt.Sprintf("exit status %d", exitError.ExitCode)
}

func convertWaitStatus(status syscall.WaitStatus) error {
	if status.Signaled() {
		return fmt.Errorf("signal: %s", status.Signal())
	}

	if status.Exited() && status.ExitStatus() != 0 {
		return &ExitError{
			ExitCode: status.ExitStatus(),
		}
	}

	return nil
}
