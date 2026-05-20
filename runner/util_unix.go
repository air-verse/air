//go:build unix && !linux

package runner

import (
	"io"
	"os"
	"os/exec"
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
		e.mainDebug("sending SIGKILL to process %d", pid)
		// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		err = syscall.Kill(-pid, syscall.SIGKILL)
		<-done // Wait for process to exit
		return
	}

	// Send SIGINT first to allow graceful shutdown
	e.mainDebug("sending interrupt to process %d", pid)
	if err = syscall.Kill(-pid, syscall.SIGINT); err != nil {
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
		// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
		err = syscall.Kill(-pid, syscall.SIGKILL)
		<-done // Wait for process to exit after SIGKILL
		return
	}
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
