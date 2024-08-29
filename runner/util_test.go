package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestFileChecksum(t *testing.T) {
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

func Test_killCmd_no_process(t *testing.T) {
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				SendInterrupt: false,
			},
		},
	}
	_, err := e.killCmd(&exec.Cmd{
		Process: &os.Process{
			Pid: 9999,
		},
	})
	if err == nil {
		t.Errorf("expected error but got none")
	}
	if !errors.Is(err, syscall.ESRCH) {
		t.Errorf("expected '%s' but got '%s'", syscall.ESRCH, errors.Unwrap(err))
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
	assert.NoError(t, err)
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

func TestGetStructureFieldTagMap(t *testing.T) {
	c := Config{}
	tagMap := flatConfig(c)
	assert.NotEmpty(t, tagMap)
	for _, i2 := range tagMap {
		fmt.Printf("%v\n", i2.fieldPath)
	}
}

func TestSetStructValue(t *testing.T) {
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "TmpDir", "asdasd")
	assert.Equal(t, "asdasd", c.TmpDir)
}

func TestNestStructValue(t *testing.T) {
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "Build.Cmd", "asdasd")
	assert.Equal(t, "asdasd", c.Build.Cmd)
}

func TestNestStructArrayValue(t *testing.T) {
	c := Config{}
	v := reflect.ValueOf(&c)
	setValue2Struct(v, "Build.ExcludeDir", "dir1,dir2")
	assert.Equal(t, []string{"dir1", "dir2"}, c.Build.ExcludeDir)
}

func TestNestStructArrayValueOverride(t *testing.T) {
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
	e := Engine{
		config: &Config{
			Build: cfgBuild{
				IncludeFile: []string{"main.go"},
			},
		},
	}
	assert.Equal(t, e.checkIncludeFile("main.go"), true)
	assert.Equal(t, e.checkIncludeFile("no.go"), false)
	assert.Equal(t, e.checkIncludeFile("."), false)
}
