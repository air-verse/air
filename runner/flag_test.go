package runner

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlag(t *testing.T) {
	// table driven tests
	type testCase struct {
		name     string
		args     []string
		expected string
		key      string
	}
	testCases := []testCase{
		{
			name:     "test1",
			args:     []string{"--build.cmd", "go build -o ./tmp/main ."},
			expected: "go build -o ./tmp/main .",
			key:      "build.cmd",
		},
		{
			name:     "tmp dir test",
			args:     []string{"--tmp_dir", "test"},
			expected: "test",
			key:      "tmp_dir",
		},
		{
			name:     "check bool",
			args:     []string{"--build.exclude_unchanged", "true"},
			expected: "true",
			key:      "build.exclude_unchanged",
		},
		{
			name:     "check exclude_regex",
			args:     []string{"--build.exclude_regex", `["_test.go"]`},
			expected: `["_test.go"]`,
			key:      "build.exclude_regex",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flag := flag.NewFlagSet(t.Name(), flag.ExitOnError)
			cmdArgs := CreateArgsFlags(flag)
			flag.Parse(tc.args)
			assert.Equal(t, tc.expected, *cmdArgs[tc.key].Value)
		})
	}
}
