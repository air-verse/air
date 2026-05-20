package runner

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestLogFuncWritesToStderr(t *testing.T) {
	t.Parallel()

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}
	os.Stderr = wErr

	// Capture stdout
	oldStdout := os.Stdout
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = wOut

	logFn := newLogFunc(rawColor, cfgLog{})
	logFn("test message from air")

	wErr.Close()
	wOut.Close()
	os.Stderr = oldStderr
	os.Stdout = oldStdout

	stderrOut, err := io.ReadAll(rErr)
	if err != nil {
		t.Fatalf("failed to read stderr: %v", err)
	}
	stdoutOut, err := io.ReadAll(rOut)
	if err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	if !strings.Contains(string(stderrOut), "test message from air") {
		t.Errorf("expected log output on stderr, got: %q", stderrOut)
	}
	if strings.Contains(string(stdoutOut), "test message from air") {
		t.Errorf("log output should not appear on stdout, got: %q", stdoutOut)
	}
}
