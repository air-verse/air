//go:build windows

package runner

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	// On Windows, SIGINT is not supported for process trees.
	// Windows uses different process termination mechanisms than Unix.
	// TASKKILL is the proper way to terminate process hierarchies on Windows.
	if e.config.Build.SendInterrupt {
		e.mainLog("send_interrupt is not supported on Windows, using TASKKILL instead")
	}

	// Use TASKKILL to forcefully terminate the entire process tree
	e.mainDebug("sending TASKKILL to process tree")
	killCmd := exec.Command("TASKKILL", "/F", "/T", "/PID", strconv.Itoa(pid))

	// Hide the console window for cleaner UX
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	err = killCmd.Run()

	// Wait for the process to fully terminate and release resources
	_, _ = cmd.Process.Wait()

	return pid, err
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error

	if !strings.Contains(cmd, ".exe") && !strings.Contains(cmd, ".bat") && !strings.Contains(cmd, ".cmd") {
		e.mainDebug("command may not be recognized as executable: %s", cmd)
	}

	// Use cmd.exe instead of PowerShell for better performance
	c := exec.Command("cmd", "/C", cmd)

	// Hide the console window
	c.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Start()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to start command: %w", err)
	}

	return c, stdout, stderr, nil
}
