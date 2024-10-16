package runner

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

func (e *Engine) modifyEnvironment(c *exec.Cmd) error {
	if e.config.Build.EnvironmentFile == "" {
		return nil
	}

	env := os.Environ()

	envFile, err := os.Open(e.config.Build.EnvironmentFile)
	if err != nil {
		return fmt.Errorf("failed to open environment file: %w")
	}
	defer envFile.Close()

	scan := bufio.NewScanner(envFile)
	for scan.Scan() {
		env = append(env, scan.Text())
	}

	c.Env = env
	return nil
}
