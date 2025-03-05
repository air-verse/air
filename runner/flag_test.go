package runner

import (
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:     "check int",
			args:     []string{"--build.kill_delay", "1000"},
			expected: "1000",
			key:      "build.kill_delay",
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
			cmdArgs := ParseConfigFlag(flag)
			require.NoError(t, flag.Parse(tc.args))
			assert.Equal(t, tc.expected, *cmdArgs[tc.key].Value)
		})
	}
}

func TestConfigRuntimeArgs(t *testing.T) {
	// table driven tests
	type testCase struct {
		name  string
		args  []string
		key   string
		check func(t *testing.T, conf *Config)
	}
	testCases := []testCase{
		{
			name: "test1",
			args: []string{"--build.cmd", "go build -o ./tmp/main ."},
			key:  "build.cmd",
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, "go build -o ./tmp/main .", conf.Build.Cmd)
			},
		},
		{
			name: "tmp dir test",
			args: []string{"--tmp_dir", "test"},
			key:  "tmp_dir",
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, "test", conf.TmpDir)
			},
		},
		{
			name: "check int64",
			args: []string{"--build.kill_delay", "1000"},
			key:  "build.kill_delay",
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, time.Duration(1000), conf.Build.KillDelay)
			},
		},
		{
			name: "check bool",
			args: []string{"--build.exclude_unchanged", "true"},
			key:  "build.exclude_unchanged",
			check: func(t *testing.T, conf *Config) {
				assert.True(t, conf.Build.ExcludeUnchanged)
			},
		},
		{
			name: "check exclude_regex",
			args: []string{"--build.exclude_regex", "_test.go,.html"},
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, []string{"_test.go", ".html"}, conf.Build.ExcludeRegex)
			},
		},
		{
			name: "check exclude_regex with empty string",
			args: []string{"--build.exclude_regex", ""},
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, []string{}, conf.Build.ExcludeRegex)
				t.Logf("%+v", conf.Build.ExcludeDir)
				assert.NotEqual(t, []string{}, conf.Build.ExcludeDir)
			},
		},
		{
			name: "check full_bin",
			args: []string{"--build.full_bin", "APP_ENV=dev APP_USER=air ./tmp/main"},
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, "APP_ENV=dev APP_USER=air ./tmp/main", conf.Build.Bin)
			},
		},

		{
			name: "check exclude_regex patterns compiled",
			args: []string{"--build.exclude_regex", "test_pattern\\.go"},
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, []string{"test_pattern\\.go"}, conf.Build.ExcludeRegex)
				patterns, err := conf.Build.RegexCompiled()
				require.NoError(t, err)
				require.NotNil(t, patterns)
				require.Len(t, patterns, 1)
				assert.True(t, patterns[0].MatchString("test_pattern.go"), "regex should match test_pattern.go")
				assert.False(t, patterns[0].MatchString("other_file.go"), "regex shouldn't match other_file.go")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.Chdir(dir))
			flag := flag.NewFlagSet(t.Name(), flag.ExitOnError)
			cmdArgs := ParseConfigFlag(flag)
			_ = flag.Parse(tc.args)
			cfg, err := InitConfig("", cmdArgs)
			if err != nil {
				log.Fatal(err)
				return
			}
			tc.check(t, cfg)
		})
	}
}
