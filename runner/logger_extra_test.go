package runner

import (
	"bytes"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureColorOutput redirects the writer fatih/color logs to, so colored log
// lines can be asserted on instead of leaking into the test output.
func captureColorOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := color.Error
	origNoColor := color.NoColor
	color.Error = &buf
	color.NoColor = true
	t.Cleanup(func() {
		color.Error = orig
		color.NoColor = origNoColor
	})
	return &buf
}

func TestGetColorFallsBackToWhite(t *testing.T) {
	t.Parallel()

	assert.Equal(t, color.FgRed, getColor("red"))
	assert.Equal(t, color.FgWhite, getColor("white"))
	assert.Equal(t, color.FgWhite, getColor("not-a-color"))
}

func TestGetLoggerFallsBackToRawLogger(t *testing.T) {
	// Not parallel: captureColorOutput swaps package-level color state.
	buf := captureColorOutput(t)

	cfg := defaultConfig()
	l := newLogger(&cfg)
	require.NotNil(t, l)

	// Known names come from the configured loggers.
	assert.NotNil(t, l.main())
	assert.NotNil(t, l.build())
	assert.NotNil(t, l.runner())
	assert.NotNil(t, l.watcher())

	// An unknown name falls back to the raw logger, which still logs.
	l.getLogger("no-such-logger")("fallback message")
	assert.NotNil(t, rawLogger())
	assert.Empty(t, buf.String(), "raw logger should not write through color.Error")
}

func TestNewLogFuncSilentSkipsOutput(t *testing.T) {
	buf := captureColorOutput(t)

	logFn := newLogFunc("white", cfgLog{Silent: true})
	logFn("should not be printed")

	assert.Empty(t, buf.String())
}

func TestNewLogFuncSkipsEmptyMessage(t *testing.T) {
	buf := captureColorOutput(t)

	logFn := newLogFunc("white", cfgLog{})
	logFn("   \n  ")

	assert.Empty(t, buf.String())
}

func TestNewLogFuncAddsTime(t *testing.T) {
	buf := captureColorOutput(t)

	logFn := newLogFunc("white", cfgLog{AddTime: true})
	logFn("hello %s", "air")

	out := buf.String()
	assert.Contains(t, out, "hello air")
	assert.Regexp(t, `^\[\d{2}:\d{2}:\d{2}\] `, out)
}

func TestNewLoggerReturnsNilWithoutConfig(t *testing.T) {
	t.Parallel()

	assert.Nil(t, newLogger(nil))
}
