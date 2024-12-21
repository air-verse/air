package runner

import (
	"os"
	"os/exec"
	"strconv"
	"time"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid
	// https://stackoverflow.com/a/44551450
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(pid))

	if e.config.Build.SendInterrupt {
		if err = kill.Run(); err != nil {
			return
		}
		time.Sleep(e.config.killDelay())
	}
	err = kill.Run()
	// Wait releases any resources associated with the Process.
	_, _ = cmd.Process.Wait()
	return pid, err
}

func (e *Engine) startCmd(cmd string) (c *exec.Cmd, err error) {
	c = exec.Command("cmd", "/c", cmd)

	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	err = c.Start()
	if err != nil {
		return nil, err
	}
	return c, nil
}
