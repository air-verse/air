package runner

import (
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	if e.config.Build.SendInterrupt {
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = syscall.Kill(-pid, syscall.SIGINT); err != nil {
			e.mainDebug("trying to send signal failed %v", err)
			return
		}
		time.Sleep(e.config.Build.KillDelay * time.Millisecond)
	}
	err = killByPid(pid)
	if err != nil {
		return pid, err
	}
	// Wait releases any resources associated with the Process.
	_, err = cmd.Process.Wait()
	if err != nil {
		return pid, err
	}

	e.mainDebug("killed process pid %d successed", pid)
	return pid, nil
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	f, err := pty.Start(c)
	return c, f, f, err
}
