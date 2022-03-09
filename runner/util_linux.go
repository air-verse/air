package runner

import (
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	// https://groups.google.com/g/golang-nuts/c/XoQ3RhFBJl8
	// only use (p *Process) Kill() will just kill the process, but it won't also the child process in linux
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return pgid, err
	}

	if e.config.Build.SendInterrupt {
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = syscall.Kill(-pgid, syscall.SIGINT); err != nil {
			return
		}
		time.Sleep(e.config.Build.KillDelay * time.Millisecond)
	}

	e.mainDebug("got pgid %v", pgid)
	if err = syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		return pgid, err
	}
	e.mainDebug("killed process pid %d successed", pgid)
	return
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	f, err := pty.Start(c)
	return c, f, f, err
}
