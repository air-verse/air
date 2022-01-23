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

	if e.config.Build.SendInterrupt {
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = syscall.Kill(-pid, syscall.SIGINT); err != nil {
			return
		}
		time.Sleep(e.config.Build.KillDelay * time.Millisecond)
	}

	// https://groups.google.com/g/golang-nuts/c/XoQ3RhFBJl8
	// only use (p *Process) Kill() will just kill the process, but it won't also the child process in linux
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		return pgid, err
	}
	e.mainDebug("got pgid %v", pgid)
	if err = syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		return pgid, err
	}
	// Wait releases any resources associated with the Process.
	_, err = cmd.Process.Wait()
	if err != nil {
		return pid, err
	}
	e.mainDebug("killed process pid %d successed", pid)
	return
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Start()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return c, stdin, stdout, stderr, nil
}
