package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildBinWithArgs tests that Build.Bin can contain embedded arguments
// for backwards compatibility (issue with v1.62.0)
func TestBuildBinWithArgs(t *testing.T) {
	tests := []struct {
		name         string
		buildBin     string
		expectedBin  string
		expectedArgs []string
	}{
		{
			name:         "Binary path only",
			buildBin:     "/go/src/github.com/orgname/appname",
			expectedBin:  "/go/src/github.com/orgname/appname",
			expectedArgs: nil,
		},
		{
			name:         "Binary with command argument (legacy case)",
			buildBin:     "/go/src/github.com/orgname/appname cmdname",
			expectedBin:  "/go/src/github.com/orgname/appname",
			expectedArgs: []string{"cmdname"},
		},
		{
			name:         "Binary with multiple arguments",
			buildBin:     "./tmp/main serve --port=8080",
			expectedBin:  "./tmp/main",
			expectedArgs: []string{"serve", "--port=8080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binPath, args := splitBinArgs(tt.buildBin)
			assert.Equal(t, tt.expectedBin, binPath)
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}
