package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDirRootPath(t *testing.T) {
	result := isDir(".")
	if result != true {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}
}

func TestIsDirMainFile(t *testing.T) {
	result := isDir("main.go")
	if result != false {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}
}

func TestIsDirFileNot(t *testing.T) {
	result := isDir("main.go")
	if result != false {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}
}

func TestExpandPathWithDot(t *testing.T) {
	path, _ := expandPath(".")
	wd, _ := os.Getwd()
	if path != wd {
		t.Errorf("expected '%s' but got '%s'", wd, path)
	}
}

func TestExpandPathWithHomePath(t *testing.T) {
	path := "~/.conf"
	result, _ := expandPath(path)
	home := os.Getenv("HOME")
	want := home + path[1:]
	if result != want {
		t.Errorf("expected '%s' but got '%s'", want, result)
	}
}

func TestNormalizeIncludeDirOutsideRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	parent := filepath.Dir(root)
	external := filepath.Join(parent, "pkg")

	cfg := &Config{
		Root: root,
		Build: cfgBuild{
			IncludeDir: []string{"../pkg"},
		},
	}
	cfg.Build.normalizeIncludeDirs(cfg.Root)

	require.Empty(t, cfg.Build.includeDirAbs)
	require.Equal(t, []string{filepath.Clean(external)}, cfg.Build.extraIncludeDirs)

	engine := &Engine{config: cfg}
	isIn, walk := engine.checkIncludeDir(filepath.Join(root, "runner"))
	require.True(t, isIn)
	require.True(t, walk)
}

func TestCheckIncludeDirRestrictsWithinRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runnerDir := filepath.Join(root, "runner")
	require.NoError(t, os.Mkdir(runnerDir, 0o755))
	otherDir := filepath.Join(root, "other")
	require.NoError(t, os.Mkdir(otherDir, 0o755))

	cfg := &Config{
		Root: root,
		Build: cfgBuild{
			IncludeDir: []string{"runner"},
		},
	}
	cfg.Build.normalizeIncludeDirs(cfg.Root)

	engine := &Engine{config: cfg}
	isIn, walk := engine.checkIncludeDir(runnerDir)
	require.True(t, isIn)
	require.True(t, walk)

	isIn, walk = engine.checkIncludeDir(otherDir)
	require.False(t, isIn)
	require.False(t, walk)
}

func TestFileChecksum(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                  string
		fileContents          []byte
		expectedChecksum      string
		expectedChecksumError string
	}{
		{
			name:                  "empty",
			fileContents:          []byte(``),
			expectedChecksum:      "",
			expectedChecksumError: "empty file, forcing rebuild without updating checksum",
		},
		{
			name:                  "simple",
			fileContents:          []byte(`foo`),
			expectedChecksum:      "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
			expectedChecksumError: "",
		},
		{
			name:                  "binary",
			fileContents:          []byte{0xF}, // invalid UTF-8 codepoint
			expectedChecksum:      "dc0e9c3658a1a3ed1ec94274d8b19925c93e1abb7ddba294923ad9bde30f8cb8",
			expectedChecksumError: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "")
			if err != nil {
				t.Fatalf("couldn't create temp file for test: %v", err)
			}

			defer func() {
				if err := f.Close(); err != nil {
					t.Errorf("error closing temp file: %v", err)
				}
				if err := os.Remove(f.Name()); err != nil {
					t.Errorf("error removing temp file: %v", err)
				}
			}()

			_, err = f.Write(test.fileContents)
			if err != nil {
				t.Fatalf("couldn't write to temp file for test: %v", err)
			}

			checksum, err := fileChecksum(f.Name())
			if err != nil && err.Error() != test.expectedChecksumError {
				t.Errorf("expected '%s' but got '%s'", test.expectedChecksumError, err.Error())
			}

			if checksum != test.expectedChecksum {
				t.Errorf("expected '%s' but got '%s'", test.expectedChecksum, checksum)
			}
		})
	}
}

func TestChecksumMap(t *testing.T) {
	t.Parallel()
	m := &checksumMap{m: make(map[string]string, 3)}

	if !m.updateFileChecksum("foo.txt", "abcxyz") {
		t.Errorf("expected no entry for foo.txt, but had one")
	}

	if m.updateFileChecksum("foo.txt", "abcxyz") {
		t.Errorf("expected matching entry for foo.txt")
	}

	if !m.updateFileChecksum("foo.txt", "123456") {
		t.Errorf("expected matching entry for foo.txt")
	}

	if !m.updateFileChecksum("bar.txt", "123456") {
		t.Errorf("expected no entry for bar.txt, but had one")
	}
}

func TestAdaptToVariousPlatforms(t *testing.T) {
	t.Parallel()
	config := &Config{
		Build: cfgBuild{
			Bin: "tmp\\main.exe  -dev",
		},
	}
	adaptToVariousPlatforms(config)
	if config.Build.Bin != "tmp\\main.exe  -dev" {
		t.Errorf("expected '%s' but got '%s'", "tmp\\main.exe  -dev", config.Build.Bin)
	}
}

func Test_killCmd_SendInterrupt_false(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)

	// Root folder of this project
	dir := filepath.Dir(b)
	err := os.Chdir(dir)
	if err != nil {
		t.Fatalf("couldn't change directory: %v", err)
	}

	// clean file before test
	os.Remove("pid")
	defer os.Remove("pid")
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: false,
			},
		},
	}
	startChan := make(chan struct {
		pid int
		cmd *exec.Cmd
	})
	go func() {
		cmd, _, _, err := e.startCmd("sh _testdata/run-many-processes.sh")
		if err != nil {
			t.Errorf("failed to start command: %v", err)
			return
		}
		pid := cmd.Process.Pid
		t.Logf("process pid is %v", pid)
		startChan <- struct {
			pid int
			cmd *exec.Cmd
		}{pid: pid, cmd: cmd}
		if err := cmd.Wait(); err != nil {
			t.Logf("failed to wait command: %v", err)
		}
		t.Logf("wait finished")
	}()
	resp := <-startChan
	t.Logf("process started. checking pid %v", resp.pid)
	time.Sleep(2 * time.Second)
	t.Logf("%v", resp.cmd.Process.Pid)
	pid, _ := e.killCmd(resp.cmd)
	t.Logf("%v was been killed", pid)
	// check processes were being killed
	// read pids from file
	bytesRead, err := os.ReadFile("pid")
	require.NoError(t, err)
	lines := strings.Split(string(bytesRead), "\n")
	for _, line := range lines {
		_, err := strconv.Atoi(line)
		if err != nil {
			t.Logf("failed to convert str to int %v", err)
			continue
		}
		_, err = exec.Command("ps", "-p", line, "-o", "comm= ").Output()
		if err == nil {
			t.Fatalf("process should be killed %v", line)
		}
	}
}

func Test_killCmd_KillsDetachedChildren(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("requires /proc")
	}

	_, b, _, _ := runtime.Caller(0)
	dir := filepath.Dir(b)
	err := os.Chdir(dir)
	if err != nil {
		t.Fatalf("couldn't change directory: %v", err)
	}

	_ = os.Remove("pid")
	defer os.Remove("pid")

	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: false,
			},
		},
	}

	startChan := make(chan *exec.Cmd)
	go func() {
		cmd, _, _, err := e.startCmd("sh _testdata/run-detached-process.sh")
		if err != nil {
			t.Errorf("failed to start command: %v", err)
			return
		}
		startChan <- cmd
		if err := cmd.Wait(); err != nil {
			t.Logf("failed to wait command: %v", err)
		}
	}()

	cmd := <-startChan
	time.Sleep(2 * time.Second)

	if _, err := e.killCmd(cmd); err != nil {
		t.Fatalf("failed to kill command: %v", err)
	}

	bytesRead, err := os.ReadFile("pid")
	require.NoError(t, err)
	lines := strings.Split(string(bytesRead), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := strconv.Atoi(line); err != nil {
			t.Logf("failed to convert str to int %v", err)
			continue
		}
		_, err = exec.Command("ps", "-p", line, "-o", "comm= ").Output()
		if err == nil {
			t.Fatalf("process should be killed %v", line)
		}
	}
}

func TestGetStructureFieldTagMap(t *testing.T) {
	t.Parallel()
	c := Config{}
	tagMap := flatConfig(c)
	assert.NotEmpty(t, tagMap)
	for _, i2 := range tagMap {
		fmt.Printf("%v\n", i2.fieldPath)
	}
}

func TestSetStructValue(t *testing.T) {
	t.Parallel()
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "TmpDir", "asdasd")
	assert.Equal(t, "asdasd", c.TmpDir)
}

func TestNestStructValue(t *testing.T) {
	t.Parallel()
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "Build.Cmd", "asdasd")
	assert.Equal(t, "asdasd", c.Build.Cmd)
}

func TestNestStructArrayValue(t *testing.T) {
	t.Parallel()
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "Build.ExcludeDir", "dir1,dir2")
	assert.Equal(t, []string{"dir1", "dir2"}, c.Build.ExcludeDir)
}

func TestNestStructArrayValueOverride(t *testing.T) {
	t.Parallel()
	c := Config{
		Build: cfgBuild{
			ExcludeDir: []string{"default1", "default2"},
		},
	}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "Build.ExcludeDir", "dir1,dir2")
	assert.Equal(t, []string{"dir1", "dir2"}, c.Build.ExcludeDir)
}

func TestCheckIncludeFile(t *testing.T) {
	t.Parallel()
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				IncludeFile: []string{"main.go"},
			},
		},
	}
	assert.True(t, e.checkIncludeFile("main.go"))
	assert.False(t, e.checkIncludeFile("no.go"))
	assert.False(t, e.checkIncludeFile("."))
}

func TestIsIncludeExt(t *testing.T) {
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				IncludeExt: []string{"go", "html"},
			},
		},
	}
	assert.True(t, e.isIncludeExt("main.go"))
	assert.True(t, e.isIncludeExt("/path/to/file.html"))
	assert.False(t, e.isIncludeExt("main.js"))
	assert.False(t, e.isIncludeExt("file"))
}

func TestIsIncludeExtWildcard(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "tmp", "main")

	e := Engine{
		config: &Config{
			Root: tmpDir,
			Build: cfgBuild{
				IncludeExt: []string{"*"},
				Entrypoint: entrypoint{binPath},
			},
		},
	}
	// Wildcard should match all file extensions
	assert.True(t, e.isIncludeExt("main.go"))
	assert.True(t, e.isIncludeExt("/path/to/file.html"))
	assert.True(t, e.isIncludeExt("main.js"))
	assert.True(t, e.isIncludeExt("file.css"))
	assert.True(t, e.isIncludeExt("file"))           // files without extension
	assert.True(t, e.isIncludeExt("/path/noext"))    // files without extension
	assert.False(t, e.isIncludeExt(binPath))         // binary file should be excluded
	assert.True(t, e.isIncludeExt("some/other/bin")) // other files without extension are ok
}

func TestIsIncludeExtWildcardWithSpaces(t *testing.T) {
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				IncludeExt: []string{" * "},
				Entrypoint: entrypoint{"/tmp/main"},
			},
		},
	}
	// Wildcard with spaces should still work
	assert.True(t, e.isIncludeExt("main.go"))
	assert.True(t, e.isIncludeExt("file.html"))
}

func TestIsBinPath(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "tmp", "main")

	e := Engine{
		config: &Config{
			Root: tmpDir,
			Build: cfgBuild{
				Entrypoint: entrypoint{binPath},
			},
		},
	}

	// Test matching path returns true
	assert.True(t, e.isBinPath(binPath))
	// Test non-matching paths return false
	assert.False(t, e.isBinPath(filepath.Join(tmpDir, "other", "file")))
	assert.False(t, e.isBinPath("unrelated.go"))
}

func TestIsBinPathEmptyBinPath(t *testing.T) {
	// Test when binPath is empty (no entrypoint configured)
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				Entrypoint: entrypoint{}, // empty entrypoint
			},
		},
	}

	// Should return false when binPath is empty
	assert.False(t, e.isBinPath("/some/path"))
	assert.False(t, e.isBinPath("main.go"))
}

func TestJoinPathRelative(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs("test")

	if err != nil {
		t.Fatalf("couldn't get absolute path for testing: %v", err)
	}

	result := joinPath(root, "x")

	assert.Equal(t, result, filepath.Join(root, "x"))
}

func TestJoinPathAbsolute(t *testing.T) {
	root, err := filepath.Abs("test")

	if err != nil {
		t.Fatalf("couldn't get absolute path for testing: %v", err)
	}

	path, err := filepath.Abs("x")

	if err != nil {
		t.Fatalf("couldn't get absolute path for testing: %v", err)
	}

	result := joinPath(root, path)

	assert.Equal(t, result, path)
}

func TestFormatPath(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name     string
		path     string
		expected string
	}

	runTests := func(t *testing.T, tests []testCase) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := formatPath(tt.path)
				if result != tt.expected {
					t.Errorf("formatPath(%q) = %q, want %q", tt.path, result, tt.expected)
				}
			})
		}
	}

	t.Run("PathPlatformSpecific", func(t *testing.T) {
		if runtime.GOOS == PlatformWindows {
			// Windows-specific tests
			tests := []testCase{
				{
					name:     "Windows style absolute path with spaces",
					path:     `C:\My Documents\My Project\tmp\app.exe`,
					expected: `& "C:\My Documents\My Project\tmp\app.exe"`,
				},
				{
					name:     "Windows style relative path with spaces",
					path:     `My Project\tmp\app.exe`,
					expected: `My Project\tmp\app.exe`,
				},
				{
					name:     "Windows style absolute path without spaces",
					path:     `C:\Documents\Project\tmp\app.exe`,
					expected: `C:\Documents\Project\tmp\app.exe`,
				},
			}
			runTests(t, tests)
		} else {
			// Unix-specific tests
			tests := []testCase{
				{
					name:     "Unix style absolute path with spaces",
					path:     `/usr/local/my project/tmp/main`,
					expected: `"/usr/local/my project/tmp/main"`,
				},
				{
					name:     "Unix style relative path with spaces",
					path:     "./my project/tmp/main",
					expected: "./my project/tmp/main",
				},
				{
					name:     "Unix style absolute path without spaces",
					path:     `/usr/local/project/tmp/main`,
					expected: `/usr/local/project/tmp/main`,
				},
			}
			runTests(t, tests)
		}
	})

	t.Run("CommonCases", func(t *testing.T) {
		tests := []testCase{
			{
				name:     "Empty path",
				path:     "",
				expected: "",
			},
			{
				name:     "Simple path",
				path:     "main.go",
				expected: "main.go",
			},
			{
				name:     "TestShouldIncludeIncludedFile",
				path:     "sh main.sh",
				expected: "sh main.sh",
			},
		}
		runTests(t, tests)
	})
}

// Test_killCmd_SendInterrupt_FastGracefulExit is a regression test for issue #671.
// It verifies that when a process exits quickly after receiving SIGINT, Air detects
// this and returns immediately instead of waiting the full kill_delay.
//
// This optimization was implemented in commit 4d26204 (2024-11-01) by Isak Styf.
// Before the fix, Air would always sleep for the full kill_delay (wasting time).
// After the fix, Air uses goroutines to detect process exit and returns early.
//
// Related: https://github.com/air-verse/air/issues/671
func Test_killCmd_SendInterrupt_FastGracefulExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("send_interrupt not supported on windows")
	}

	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: true,
				KillDelay:     2 * time.Second, // Set high to make the waste observable
			},
		},
	}

	// Process that exits immediately on SIGINT
	// trap "exit 0" INT means: exit cleanly when receiving SIGINT
	// sleep 100 means: if no signal, run for 100 seconds
	cmd, _, _, err := e.startCmd(`sh -c 'trap "exit 0" INT; sleep 100'`)
	require.NoError(t, err, "failed to start command")

	// Give the process a moment to start and set up the signal handler
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	pid, err := e.killCmd(cmd)
	elapsed := time.Since(start)

	require.NoError(t, err, "killCmd should succeed")
	assert.Positive(t, pid, "should return valid pid")

	// Core assertion: should complete in much less than 2 seconds
	// Process exits immediately on SIGINT, so should finish in <500ms
	// With the fix (commit 4d26204), this should PASS
	// Without the fix, this would FAIL (would take 2+ seconds)
	assert.Less(t, elapsed, 500*time.Millisecond,
		"killCmd should return quickly when process exits gracefully on SIGINT, "+
			"but took %v (expected < 500ms). Regression of issue #671!",
		elapsed)

	t.Logf("✅ PASS: Process exited gracefully in %v (kill_delay was 2s, saved ~%.1fs)",
		elapsed, 2.0-elapsed.Seconds())
}

// Test_killCmd_SendInterrupt_IgnoresSIGINT is a regression test for issue #671.
// It verifies that processes which ignore SIGINT are still killed with SIGKILL
// after kill_delay. This ensures the optimization (commit 4d26204) doesn't break
// the fallback behavior for misbehaving processes.
//
// Related: https://github.com/air-verse/air/issues/671
func Test_killCmd_SendInterrupt_IgnoresSIGINT(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("send_interrupt not supported on windows")
	}

	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: true,
				KillDelay:     500 * time.Millisecond,
			},
		},
	}

	// Process that ignores SIGINT
	// trap "" INT means: ignore SIGINT signal
	cmd, _, _, err := e.startCmd(`sh -c 'trap "" INT; sleep 100'`)
	require.NoError(t, err, "failed to start command")

	// Give the process a moment to start and set up the signal handler
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	pid, err := e.killCmd(cmd)
	elapsed := time.Since(start)

	require.NoError(t, err, "killCmd should succeed")
	assert.Positive(t, pid, "should return valid pid")

	// Should wait at least kill_delay before sending SIGKILL
	assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond,
		"killCmd should wait at least kill_delay when process ignores SIGINT")

	// But should not wait too long after SIGKILL
	assert.Less(t, elapsed, 1*time.Second,
		"killCmd should not wait too long after sending SIGKILL, "+
			"but took %v", elapsed)

	t.Logf("✅ PASS: Process ignored SIGINT, killed with SIGKILL after %v (expected behavior)", elapsed)
}

// Test_killCmd_SendInterrupt_SlowGracefulExit is a regression test for issue #671.
// It verifies that when a process takes some time to clean up after receiving
// SIGINT but still exits within kill_delay, Air returns as soon as the process exits
// (not waiting the full kill_delay).
//
// This optimization was implemented in commit 4d26204 (2024-11-01).
// Related: https://github.com/air-verse/air/issues/671
func Test_killCmd_SendInterrupt_SlowGracefulExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("send_interrupt not supported on windows")
	}

	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: true,
				KillDelay:     1 * time.Second,
			},
		},
	}

	// Process that takes 300ms to exit after SIGINT (simulating cleanup)
	// trap "sleep 0.3; exit 0" INT means: when SIGINT received, sleep 300ms then exit
	cmd, _, _, err := e.startCmd(`sh -c 'trap "sleep 0.3; exit 0" INT; sleep 100'`)
	require.NoError(t, err, "failed to start command")

	// Give the process a moment to start and set up the signal handler
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	pid, err := e.killCmd(cmd)
	elapsed := time.Since(start)

	require.NoError(t, err, "killCmd should succeed")
	assert.Positive(t, pid, "should return valid pid")

	// Should wait at least for the process cleanup time (~300ms)
	assert.Greater(t, elapsed, 250*time.Millisecond,
		"should wait at least for process cleanup time (~300ms)")

	// Should return shortly after process exits (~300-500ms)
	// With the fix (commit 4d26204), this should PASS
	// Without the fix, this would FAIL (would take 1+ seconds)
	assert.Less(t, elapsed, 600*time.Millisecond,
		"killCmd should return soon after process exits, "+
			"but took %v (expected ~300-500ms). Regression of issue #671!",
		elapsed)

	t.Logf("✅ PASS: Process exited gracefully in %v after cleanup (kill_delay was 1s, saved ~%.1fs)",
		elapsed, 1.0-elapsed.Seconds())
}

func TestIsDangerousRoot(t *testing.T) {
	t.Parallel()

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err, "failed to get user home directory")

	tests := []struct {
		name        string
		path        string
		isDangerous bool
		description string
	}{
		{
			name:        "root directory",
			path:        "/",
			isDangerous: true,
			description: "root directory (/)",
		},
		{
			name:        "root user home",
			path:        "/root",
			isDangerous: true,
			description: "/root directory",
		},
		{
			name:        "user home directory",
			path:        homeDir,
			isDangerous: true,
			description: "home directory (~)",
		},
		{
			name:        "normal project directory",
			path:        "/home/user/myproject",
			isDangerous: false,
			description: "",
		},
		{
			name:        "tmp directory",
			path:        "/tmp/test-project",
			isDangerous: false,
			description: "",
		},
		{
			name:        "current directory in project",
			path:        ".",
			isDangerous: false,
			description: "",
		},
		{
			name:        "subdirectory of home",
			path:        filepath.Join(homeDir, "projects", "myapp"),
			isDangerous: false,
			description: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDangerous, desc := isDangerousRoot(tt.path)
			assert.Equal(t, tt.isDangerous, isDangerous, "isDangerous mismatch for path %s", tt.path)
			if tt.isDangerous {
				assert.Equal(t, tt.description, desc, "description mismatch for path %s", tt.path)
			}
		})
	}
}
