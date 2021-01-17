package runner

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
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
			f, err := ioutil.TempFile("", "")
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

func TestAddArgs(t *testing.T) {
	cfg := config{
		Build: cfgBuild{
			Bin: "/tmp/main",
			ArgsBin: []string{
				"server",
			},
		},
	}

	cmd := addArgs(cfg.Build.Bin, cfg.Build.ArgsBin)
	if !strings.HasSuffix(cmd, "server") {
		t.Errorf("expected contain entry for server")
	}

	cfg.Build.ArgsBin = append(cfg.Build.ArgsBin, "--help")
	cmd = addArgs(cfg.Build.Bin, cfg.Build.ArgsBin)
	if !strings.HasSuffix(cmd, " --help") {
		t.Errorf("expected contain entry for --help")
	}

	split := strings.Split(cmd, " ")
	if len(split) != 3 {
		t.Errorf("expacted '%d' but got '%d' ", 3, len(split))
	}
}
