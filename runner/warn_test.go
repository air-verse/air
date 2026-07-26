package runner

import (
	"bytes"
	"io"
	"testing"
)

// swapWarnOut points warnOut at w until the test ends.
func swapWarnOut(t *testing.T, w io.Writer) {
	t.Helper()
	warnMu.Lock()
	orig := warnOut
	warnOut = w
	warnMu.Unlock()
	t.Cleanup(func() {
		warnMu.Lock()
		warnOut = orig
		warnMu.Unlock()
	})
}

// captureWarnings collects configuration warnings emitted during the test and
// returns them via the returned func, which may be called more than once.
//
// Tests using it must not call t.Parallel(): warnOut is process-wide, so a
// concurrent test emitting warnings would end up in this buffer.
func captureWarnings(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	swapWarnOut(t, &buf)
	return func() string {
		warnMu.Lock()
		defer warnMu.Unlock()
		return buf.String()
	}
}

// silenceWarnings discards configuration warnings (e.g. the build.bin
// deprecation notice) that would otherwise leak into test output.
func silenceWarnings(t *testing.T) {
	t.Helper()
	swapWarnOut(t, io.Discard)
}
