package runner

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// silenceStdoutStderr points the process-wide streams at os.DevNull so tests
// that copy large command output don't flood CI logs. The swap is restored on
// cleanup. Tests using it must not call t.Parallel(): os.Stdout and os.Stderr
// are process-wide, so a parallel test would see the swapped values.
func silenceStdoutStderr(t *testing.T) {
	t.Helper()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout = devnull
	os.Stderr = devnull
	t.Cleanup(func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
		require.NoError(t, devnull.Close())
	})
}

// newCaptureTestEngine builds a default-config engine rooted in a temp dir.
func newCaptureTestEngine(t *testing.T) *Engine {
	t.Helper()
	chdir(t, t.TempDir())
	e, err := NewEngine("", nil, true)
	require.NoError(t, err)
	return e
}

func TestRunCommandCopyOutputCapturesBoundedTail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix shell syntax; windows runs commands through powershell")
	}
	e := newCaptureTestEngine(t)
	silenceStdoutStderr(t)

	out, err := e.runCommandCopyOutput(`sh -c 'seq 1 200000; echo END_MARKER; exit 1'`)
	require.Error(t, err)
	assert.NotEmpty(t, out)
	assert.LessOrEqual(t, len(out), maxCapturedOutputSize)
	assert.Contains(t, out, "END_MARKER")
	// seq emits every integer exactly once, so a standalone line "1" only
	// exists at the head of the stream; its absence proves the oldest bytes
	// were dropped in favor of the tail.
	assert.NotContains(t, out, "\n1\n")
}

func TestRunCommandCopyOutputSmallSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix shell syntax; windows runs commands through powershell")
	}
	e := newCaptureTestEngine(t)

	out, err := e.runCommandCopyOutput(`echo hello-capture`)
	require.NoError(t, err)
	assert.Equal(t, "hello-capture\n", out)
}
