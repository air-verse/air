package runner

import (
	"io"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

func killCmd(cmd *exec.Cmd) (int, error) {
	pid := cmd.Process.Pid
	// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
	err := syscall.Kill(-pid, syscall.SIGKILL)

	// Wait releases any resources associated with the Process.
	_, _ = cmd.Process.Wait()

	return pid, err
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	f, err := pty.Start(c)
	return c, f, f, err
}
