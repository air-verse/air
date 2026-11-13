//go:build unix

package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverlappingBuildsLeaveOneRunningBin tests that overlapping builds leave exactly one bin
// running, the newest. This is a regression test for issue #947: build 1 was superseded by build 2
// but finished first, which let build 3 run alongside build 2, and both started a bin.
func TestOverlappingBuildsLeaveOneRunningBin(t *testing.T) {
	tmpDir := t.TempDir()
	chdir(t, tmpDir)
	engine, err := NewEngine("", nil, true)
	require.NoError(t, err)
	engine.config.Log.Silent = true
	// Counting the initial build as 0, build 1 takes 100ms and builds 2 and 3 wait for the release.
	// Every build fails, but still starts the bin because stop_on_error is off.
	engine.config.Build.Cmd = `n=$(wc -c < builds 2>/dev/null || echo 0); printf x >> builds; case $n in 1) sleep 0.1;; 2|3) [ -e released ] || cat release;; esac; false`
	engine.config.Build.StopOnError = false
	engine.config.Build.Delay = 1
	engine.config.Build.Entrypoint = entrypoint{}
	engine.config.Build.Bin = `sh -c 'echo $$ >> pids; exec sleep 60'`

	started := func() []int {
		data, _ := os.ReadFile(filepath.Join(tmpDir, "pids"))
		var pids []int
		for _, field := range strings.Fields(string(data)) {
			var pid int
			_, _ = fmt.Sscan(field, &pid)
			pids = append(pids, pid)
		}
		return pids
	}
	alive := func() []int {
		var pids []int
		for _, pid := range started() {
			if syscall.Kill(pid, 0) == nil {
				pids = append(pids, pid)
			}
		}
		return pids
	}
	t.Cleanup(func() {
		for _, pid := range alive() {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	require.NoError(t, syscall.Mkfifo(filepath.Join(tmpDir, "release"), 0o600))
	go engine.Run()
	require.NoError(t, waitForCondition(t, 5*time.Second, func() bool { return len(alive()) == 1 }, "first bin"))
	for _, wait := range []time.Duration{50, 100, 50} {
		engine.eventCh <- "main.go"
		time.Sleep(wait * time.Millisecond)
	}
	require.NoError(t, waitForCondition(t, 5*time.Second, func() bool {
		builds, _ := os.ReadFile(filepath.Join(tmpDir, "builds"))
		return len(builds) >= 4
	}, "build 3 to start"))
	time.Sleep(100 * time.Millisecond) // for build 2 to reach the release too, if builds can overlap
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "released"), nil, 0o644))
	// Opening the FIFO's write end unblocks every reader at once. It fails until a reader opens it.
	require.NoError(t, waitForCondition(t, 5*time.Second, func() bool {
		release, err := os.OpenFile(filepath.Join(tmpDir, "release"), os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err != nil {
			return false
		}
		_ = release.Close()
		return true
	}, "a build to wait for the release"))
	time.Sleep(time.Second) // for every unblocked build to finish
	pids := started()
	require.NotEmpty(t, pids)
	assert.Equal(t, pids[len(pids)-1:], alive(), "every bin but the newest should have been stopped")

	engine.Stop()
	require.NoError(t, waitForEngineState(t, engine, false, 5*time.Second))
	require.NoError(t, waitForCondition(t, 5*time.Second, func() bool { return len(alive()) == 0 }, "bins stopped with the engine"))
}
