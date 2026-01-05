package runner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	// Start a goroutine to wait for the process to exit
	done := make(chan struct{})
	go func() {
		_, _ = cmd.Process.Wait()
		close(done)
	}()

	// If not using send_interrupt, just kill immediately
	if !e.config.Build.SendInterrupt {
		e.mainDebug("sending SIGKILL to process tree")
		err = sendSignalToProcessTree(pid, syscall.SIGKILL)
		<-done // Wait for process to exit
		return
	}

	// Send SIGINT first to allow graceful shutdown
	e.mainDebug("sending interrupt to process tree")
	if err = sendSignalToProcessTree(pid, syscall.SIGINT); err != nil {
		return
	}

	killDelay := e.config.killDelay()
	e.mainDebug("waiting up to %s for graceful shutdown", killDelay.String())

	// Wait for either the process to exit gracefully or the kill delay to expire
	select {
	case <-done:
		// Process exited gracefully after SIGINT - excellent!
		e.mainDebug("process exited gracefully after SIGINT")
		return
	case <-time.After(killDelay):
		// Timeout expired, need to force kill
		e.mainDebug("kill delay expired, sending SIGKILL")
		err = sendSignalToProcessTree(pid, syscall.SIGKILL)
		<-done // Wait for process to exit after SIGKILL
		return
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

	// Send signals to descendants concurrently for better performance
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, child := range descendants {
		wg.Add(1)
		go func(childPID int) {
			defer wg.Done()
			if err := syscall.Kill(childPID, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(child)
	}
	wg.Wait()

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
