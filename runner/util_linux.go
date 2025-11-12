package runner

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	var killDelay time.Duration

	waitResult := make(chan error)
	go func() {
		defer close(waitResult)
		// ignore any error from Wait
		_, _ = cmd.Process.Wait()
	}()

	if e.config.Build.SendInterrupt {
		e.mainDebug("sending interrupt to process tree")
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = sendSignalToProcessTree(pid, syscall.SIGINT); err != nil {
			return
		}
		// the kill delay is 0 by default unless the user has configured send_interrupt=true
		// in which case it is fetched from the kill_delay setting in the .air.toml
		killDelay = e.config.killDelay()
		e.mainDebug("setting a kill timer for %s", killDelay.String())
	}

	// prepare a cancel context that can stop the killing if it is not needed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	killResult := make(chan error)
	// Spawn a goroutine that will kill the process after kill delay if we have not
	// received a wait result before that.
	go func() {
		select {
		case <-time.After(killDelay):
			e.mainDebug("kill timer expired")
			// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
			killResult <- sendSignalToProcessTree(pid, syscall.SIGKILL)
		case <-ctx.Done():
			e.mainDebug("kill timer canceled")
			return
		}
	}()

	results := make([]error, 0, 2)

	for {
		// collect the responses from the kill and wait goroutines
		select {
		case err = <-killResult:
			results = append(results, err)
		case err = <-waitResult:
			results = append(results, err)
			// if we have a kill delay, but have not received a kill
			// result yet, we fake the kill result to exit the loop below
			if killDelay > 0 && len(results) == 1 {
				results = append(results, nil)
			}
		}

		if len(results) >= 2 {
			err = errors.Join(results...)
			return
		}
	}
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	// Set Setpgid to create a new process group (not possible when using pty)
	c.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	return c, stdout, stderr, nil
}

func sendSignalToProcessTree(pid int, sig syscall.Signal) error {
	descendants, descendantsErr := collectDescendantPIDs(pid)
	var errs []error

	if descendantsErr != nil && !errors.Is(descendantsErr, os.ErrNotExist) {
		errs = append(errs, descendantsErr)
	}

	// Try to signal the whole process group first.
	groupErr := syscall.Kill(-pid, sig)
	if groupErr != nil && !errors.Is(groupErr, syscall.EPERM) && !errors.Is(groupErr, syscall.ESRCH) {
		errs = append(errs, groupErr)
	}

	// Always signal the root pid as well in case it moved to another process group.
	procErr := syscall.Kill(pid, sig)
	if procErr != nil && !errors.Is(procErr, syscall.ESRCH) {
		errs = append(errs, procErr)
	}

	for _, child := range descendants {
		if err := syscall.Kill(child, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
			errs = append(errs, err)
		}
	}

	if len(errs) == 0 && errors.Is(groupErr, syscall.ESRCH) && errors.Is(procErr, syscall.ESRCH) {
		return syscall.ESRCH
	}

	return errors.Join(errs...)
}

func collectDescendantPIDs(pid int) ([]int, error) {
	children, err := readChildPIDs(pid)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var (
		allChildren = make([]int, 0, len(children))
		errs        []error
	)
	for _, child := range children {
		allChildren = append(allChildren, child)
		descendants, err := collectDescendantPIDs(child)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			errs = append(errs, err)
			continue
		}
		allChildren = append(allChildren, descendants...)
	}

	return allChildren, errors.Join(errs...)
}

func readChildPIDs(pid int) ([]int, error) {
	path := fmt.Sprintf("/proc/%d/task/%d/children", pid, pid)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	children := make([]int, 0, len(fields))
	for _, field := range fields {
		childPID, err := strconv.Atoi(field)
		if err != nil {
			continue
		}
		children = append(children, childPID)
	}

	return children, nil
}
