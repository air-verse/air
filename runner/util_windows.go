//go:build windows

package runner

import (
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	// On Windows, SIGINT is not supported for process trees
	// Use TASKKILL to forcefully terminate the process hierarchy
	if e.config.Build.SendInterrupt {
		e.mainLog("send_interrupt is not supported on Windows, using TASKKILL instead")
	}

	// Single TASKKILL execution to avoid double-kill bug
	e.mainDebug("sending TASKKILL to process tree")
	killCmd := exec.Command("TASKKILL", "/F", "/T", "/PID", strconv.Itoa(pid))

	// Hide the taskkill console window
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}

	err = killCmd.Run()

	// Wait for process to terminate and release resources
	_, _ = cmd.Process.Wait()

	return pid, err
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error

	if !strings.Contains(cmd, ".exe") {
		e.runnerLog("CMD will not recognize non .exe file for execution, path: %s", cmd)
	}

	// Keep PowerShell to avoid cmd.exe sound issues (#707)
	// Use -NoProfile and -NonInteractive for better performance
	c := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", cmd)

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
	return c, stdout, stderr, err
}
