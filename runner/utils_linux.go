package runner

import (
	"io"
	"os/exec"
	"syscall"
)

func killCmd(cmd *exec.Cmd) (int, error) {
	pid := cmd.Process.Pid
	// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
	err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	return pid, err
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error
	c := exec.Command("/bin/sh", "-c", cmd)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	err = c.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	return c, stdout, stderr, err
}
