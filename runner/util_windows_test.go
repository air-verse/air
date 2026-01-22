package runner

import (
	"runtime"
	"strings"
	"testing"
)

func TestAdaptToVariousPlatformsFullBinWindows(t *testing.T) {
	if runtime.GOOS != PlatformWindows {
		t.Skip("windows-only behavior")
	}

	t.Parallel()

	tests := []struct {
		name     string
		fullBin  string
		expected string
	}{
		{
			name:     "exe already",
			fullBin:  `.\tmp\main.exe`,
			expected: `.\tmp\main.exe`,
		},
		{
			name:     "append exe",
			fullBin:  `.\tmp\main`,
			expected: `.\tmp\main.exe`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				Build: cfgBuild{
					FullBin: tt.fullBin,
				},
			}
			adaptToVariousPlatforms(config)
			if config.Build.FullBin != tt.expected {
				t.Fatalf("expected full_bin %q, got %q", tt.expected, config.Build.FullBin)
			}
			if strings.HasPrefix(strings.ToLower(config.Build.FullBin), "start ") {
				t.Fatalf("unexpected start prefix in full_bin: %q", config.Build.FullBin)
			}
		})
	}
}
