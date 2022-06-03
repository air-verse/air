package runner

import (
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/pkg/errors"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return pgid, errors.Wrapf(err, "failed to get pgid, pid %v", pid)
	}

	if e.config.Build.SendInterrupt {
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = syscall.Kill(-pgid, syscall.SIGINT); err != nil {
			return
		}
		time.Sleep(e.config.Build.KillDelay * time.Millisecond)
	}
	// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
	err = syscall.Kill(-pgid, syscall.SIGKILL)
	// Wait releases any resources associated with the Process.
	_, _ = cmd.Process.Wait()
	e.mainDebug("killed process pid %d successed", pgid)
	return pgid, nil
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	f, err := pty.Start(c)
	return c, f, f, err
}
