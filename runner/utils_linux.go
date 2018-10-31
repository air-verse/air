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

func (e *Engine) startCmd(cmd string, args []string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error

	params := e.cmdgen(cmd, args)

	c := exec.Command("/bin/sh", params...)

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

func (e *Engine) cmdgen(cmd string, args []string) []string {

	prms := []string{"-c", cmd}
	prms = append(prms, args...)

	return prms
}
