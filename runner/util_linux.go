package runner

import (
	"context"
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func (e *Engine) killCmd(cmd *exec.Cmd) (pid int, err error) {
	pid = cmd.Process.Pid

	var killDelay time.Duration

	if e.config.Build.SendInterrupt {
		e.mainDebug("sending interrupt to process %d", pid)
		// Sending a signal to make it clear to the process that it is time to turn off
		if err = syscall.Kill(-pid, syscall.SIGINT); err != nil {
			return
		}
		// the kill delay is 0 by default unless the user has configured send_interrupt=true
		// in which case it is fetched from the kill_delay setting in the .air.toml
		killDelay = e.config.killDelay()
		e.mainDebug("setting a kill timer for %s", killDelay.String())
	}

	waitResult := make(chan error)
	go func() {
		defer close(waitResult)
		_, _ = cmd.Process.Wait()
	}()

	// prepare a cancel context that can stop the killing if it is not needed
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	killResult := make(chan error)
	// Spawn a goroutine that will kill the process after kill delay if we have not
	// received a wait result before that.
	go func() {
		select {
		case <-time.After(killDelay):
			// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
			killResult <- syscall.Kill(-pid, syscall.SIGKILL)
		case <-ctx.Done():
			return
		}
	}()

	for {
		select {
		case err = <-killResult:
		case <-waitResult:
			return
		}
	}
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	c := exec.Command("/bin/sh", "-c", cmd)
	f, err := pty.Start(c)
	return c, f, f, err
}
