// Package runner …
package runner

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func chdir(t *testing.T, targetDir string) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to getwd: %v", err)
	}
	if err := os.Chdir(targetDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})
}

// waitForCondition waits for a condition to be true with fast polling.
// Uses environment-aware timeout multiplier for CI compatibility.
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, description string) error {
	t.Helper()

	// CI environments may be slower, use 2x timeout
	timeoutMultiplier := 1.0
	if os.Getenv("CI") != "" {
		timeoutMultiplier = 2.0
	}

	adjustedTimeout := time.Duration(float64(timeout) * timeoutMultiplier)
	deadline := time.Now().Add(adjustedTimeout)
	ticker := time.NewTicker(20 * time.Millisecond) // Fast polling: 20ms
	defer ticker.Stop()

	for {
		if condition() {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for: %s (timeout: %v)", description, adjustedTimeout)
		}
		<-ticker.C
	}
}

// waitForEngineState waits for engine to reach the specified running state.
func waitForEngineState(t *testing.T, engine *Engine, running bool, timeout time.Duration) error {
	t.Helper()
	return waitForCondition(t, timeout, func() bool {
		return engine.running.Load() == running
	}, fmt.Sprintf("engine running=%v", running))
}
