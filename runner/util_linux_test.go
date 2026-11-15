package runner

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_sendSignalToProcessTree_ConcurrentSignalSending(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires /proc")
	}

	_, b, _, _ := runtime.Caller(0)
	dir := filepath.Dir(b)
	err := os.Chdir(dir)
	if err != nil {
		t.Fatalf("couldn't change directory: %v", err)
	}

	_ = os.Remove("pid")
	defer os.Remove("pid")

	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: false,
			},
		},
	}

	// Start a process tree with multiple children
	startChan := make(chan *exec.Cmd)
	go func() {
		cmd, _, _, err := e.startCmd("sh _testdata/run-many-processes.sh")
		if err != nil {
			t.Errorf("failed to start command: %v", err)
			return
		}
		startChan <- cmd
		if err := cmd.Wait(); err != nil {
			t.Logf("wait returned: %v", err)
		}
	}()

	cmd := <-startChan
	pid := cmd.Process.Pid
	time.Sleep(2 * time.Second)

	// Send signal using the concurrent implementation
	err = sendSignalToProcessTree(pid, syscall.SIGKILL)

	// Should not return an error for successful kill
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Errorf("unexpected error from sendSignalToProcessTree: %v", err)
	}

	// Verify all processes were killed
	bytesRead, err := os.ReadFile("pid")
	require.NoError(t, err)
	lines := strings.Split(string(bytesRead), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := strconv.Atoi(line); err != nil {
			t.Logf("failed to convert str to int %v", err)
			continue
		}
		_, err = exec.Command("ps", "-p", line, "-o", "comm= ").Output()
		if err == nil {
			t.Fatalf("process should be killed %v", line)
		}
	}
}
