package runner

import (
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

	if e.config.Build.SendInterrupt {
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = sendSignalToProcessTree(pid, syscall.SIGINT); err != nil {
			return
		}
		time.Sleep(e.config.killDelay())
	}

	// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
	err = sendSignalToProcessTree(pid, syscall.SIGKILL)

	// Wait releases any resources associated with the Process.
	_, _ = cmd.Process.Wait()
	return
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	// because using pty cannot have same pgid
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
	var errs []error

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

	if err := signalDescendants(pid, sig); err != nil {
		errs = append(errs, err)
	}

	if len(errs) == 0 && errors.Is(groupErr, syscall.ESRCH) && errors.Is(procErr, syscall.ESRCH) {
		return syscall.ESRCH
	}

	return errors.Join(errs...)
}

func signalDescendants(pid int, sig syscall.Signal) error {
	children, err := readChildPIDs(pid)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	var errs []error
	for _, child := range children {
		if err := syscall.Kill(child, sig); err != nil && !errors.Is(err, syscall.ESRCH) {
			errs = append(errs, err)
		}
		if err := signalDescendants(child, sig); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
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
