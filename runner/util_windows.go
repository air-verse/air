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
	e.runnerLog("trying to kill pid %d, cmd %v", pid, cmd.Args)

	// On Windows, send_interrupt (SIGINT) is not supported
	// We always use TASKKILL to forcefully terminate the process tree
	if e.config.Build.SendInterrupt {
		e.mainLog("send_interrupt is not supported on Windows, using TASKKILL instead")
	}

	// Use TASKKILL to kill the entire process tree
	// /F = Force termination
	// /T = Terminate all child processes
	// /PID = Process ID to kill
	e.runnerLog("sending TASKKILL to process tree")
	killCmd := exec.Command("TASKKILL", "/F", "/T", "/PID", strconv.Itoa(pid))
	
	// Hide the taskkill console window for cleaner UX
	killCmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}

	if err = killCmd.Run(); err != nil {
		// Process might already be dead, which is acceptable
		e.runnerLog("taskkill returned error (process may already be terminated): %v", err)
	} else {
		e.runnerLog("cmd killed, pid: %d", pid)
	}

	// Wait for the process to fully terminate
	// This releases any resources associated with the process
	_, waitErr := cmd.Process.Wait()
	if waitErr != nil {
		e.runnerLog("wait error: %v", waitErr)
	}

	return pid, err
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error

	// Warn if command doesn't look like a Windows executable
	if !strings.Contains(cmd, ".exe") && !strings.Contains(cmd, ".bat") && !strings.Contains(cmd, ".cmd") {
		e.runnerLog("warning: command may not be recognized as executable: %s", cmd)
	}

	// Use cmd.exe instead of PowerShell for better performance
	// PowerShell has significant startup overhead
	c := exec.Command("cmd", "/C", cmd)
	
	// Hide the cmd.exe console window
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
