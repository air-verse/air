package runner

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	bin = `./tmp/main`
	cmd = "go build -o ./tmp/main ."
)

func getWindowsConfig() Config {
	build := cfgBuild{
		PreCmd:       []string{"echo Hello Air"},
		Cmd:          "go build -o ./tmp/main .",
		Bin:          "./tmp/main",
		Log:          "build-errors.log",
		IncludeExt:   []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir:   []string{"assets", "tmp", "vendor", "testdata"},
		ExcludeRegex: []string{"_test.go"},
		Delay:        1000,
		StopOnError:  true,
	}
	if runtime.GOOS == "windows" {
		build.Bin = bin
		build.Cmd = cmd
	}

	return Config{
		Root:        ".",
		TmpDir:      "tmp",
		TestDataDir: "testdata",
		Build:       build,
	}
}

func TestBinCmdPath(t *testing.T) {
	var err error

	c := getWindowsConfig()
	err = c.preprocess(nil)
	if err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {

		if !strings.HasSuffix(c.Build.Bin, "exe") {
			t.Fail()
		}

		if !strings.Contains(c.Build.Bin, "exe") {
			t.Fail()
		}
	} else {

		if strings.HasSuffix(c.Build.Bin, "exe") {
			t.Fail()
		}

		if strings.Contains(c.Build.Bin, "exe") {
			t.Fail()
		}
	}
}

func TestDefaultPathConfig(t *testing.T) {
	tests := []struct {
		name string
		path string
		root string
	}{{
		name: "Invalid Path",
		path: "invalid/path",
		root: ".",
	}, {
		name: "TOML",
		path: "_testdata/toml",
		root: "toml_root",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(airWd, tt.path)
			c, err := defaultPathConfig()
			if err != nil {
				t.Fatalf("Should not be fail: %s.", err)
			}

			if got, want := c.Root, tt.root; got != want {
				t.Fatalf("Root is %s, but want %s.", got, want)
			}
		})
	}
}

func TestReadConfByName(t *testing.T) {
	_ = os.Unsetenv(airWd)
	config, _ := readConfByName(dftTOML)
	if config != nil {
		t.Fatalf("expect Config is nil,but get a not nil Config")
	}
}

func TestDefaultStopOnError(t *testing.T) {
	df := defaultConfig()
	if !df.Build.StopOnError {
		t.Fatal("expected StopOnError to default to true")
	}
}

func TestConfPreprocess(t *testing.T) {
	t.Setenv(airWd, "_testdata/toml")
	df := defaultConfig()
	err := df.preprocess(nil)
	if err != nil {
		t.Fatalf("preprocess error %v", err)
	}
	suffix := "/_testdata/toml/tmp/main"
	binPath := df.Build.Entrypoint.binary()
	if !strings.HasSuffix(binPath, suffix) {
		t.Fatalf("bin path is %s, but not have suffix  %s.", binPath, suffix)
	}
}

func TestEntrypointResolvesAbsolutePath(t *testing.T) {
	base := t.TempDir()
	rootWithSpace := filepath.Join(base, "with space")
	if err := os.MkdirAll(filepath.Join(rootWithSpace, "tmp"), 0o755); err != nil {
		t.Fatalf("failed to prepare tmp dir: %v", err)
	}

	cfg := defaultConfig()
	cfg.Root = rootWithSpace
	cfg.Build.Entrypoint = entrypoint{"./tmp/main"}

	if err := cfg.preprocess(nil); err != nil {
		t.Fatalf("preprocess error %v", err)
	}

	want := filepath.Join(rootWithSpace, "tmp", "main")
	if got := cfg.Build.Entrypoint.binary(); got != want {
		t.Fatalf("entrypoint is %s, but want %s", got, want)
	}

	if cfg.binPath() != want {
		t.Fatalf("bin path is %s, but want %s", cfg.binPath(), want)
	}
}

func TestEntrypointResolvesFromPath(t *testing.T) {
	root := t.TempDir()
	pathDir := t.TempDir()

	binName := "air-entrypoint-path"
	fileName := binName
	fileContents := "#!/bin/sh\nexit 0\n"
	if runtime.GOOS == "windows" {
		fileName += ".bat"
		fileContents = "@echo off\r\n"
		t.Setenv("PATHEXT", ".BAT;.EXE")
	}
	fullPath := filepath.Join(pathDir, fileName)
	if err := os.WriteFile(fullPath, []byte(fileContents), 0o755); err != nil {
		t.Fatalf("failed to write fake binary: %v", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(fullPath, 0o755); err != nil {
			t.Fatalf("failed to make fake binary executable: %v", err)
		}
	}

	t.Setenv("PATH", pathDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := defaultConfig()
	cfg.Root = root
	cfg.Build.Entrypoint = entrypoint{binName}

	if err := cfg.preprocess(nil); err != nil {
		t.Fatalf("preprocess error %v", err)
	}

	want := fullPath
	if got := cfg.Build.Entrypoint.binary(); got != want {
		t.Fatalf("entrypoint resolved to %s, want %s", got, want)
	}
}

func TestEntrypointPreservesArgs(t *testing.T) {
	root := t.TempDir()
	cfg := defaultConfig()
	cfg.Root = root
	cfg.Build.Entrypoint = entrypoint{"./tmp/main", "server", ":8080"}

	if err := cfg.preprocess(nil); err != nil {
		t.Fatalf("preprocess error %v", err)
	}

	wantBin := filepath.Join(root, "tmp", "main")
	if cfg.Build.Entrypoint.binary() != wantBin {
		t.Fatalf("entrypoint binary is %s, want %s", cfg.Build.Entrypoint.binary(), wantBin)
	}

	wantArgs := []string{"server", ":8080"}
	if got := cfg.Build.Entrypoint.args(); !reflect.DeepEqual(got, wantArgs) {
		t.Fatalf("entrypoint args mismatch, got %v want %v", got, wantArgs)
	}
}

func TestConfigWithRuntimeArgs(t *testing.T) {
	runtimeArg := "-flag=value"

	// inject runtime arg
	oldArgs := os.Args
	defer func() {
		os.Args = oldArgs
		flag.Parse()
	}()
	os.Args = []string{"air", "--", runtimeArg}
	flag.Parse()

	t.Run("when using bin", func(t *testing.T) {
		df := defaultConfig()
		if err := df.preprocess(nil); err != nil {
			t.Fatalf("preprocess error %v", err)
		}

		if !contains(df.Build.ArgsBin, runtimeArg) {
			t.Fatalf("missing expected runtime arg: %s", runtimeArg)
		}
	})

	t.Run("when using full_bin", func(t *testing.T) {
		df := defaultConfig()
		df.Build.FullBin = "./tmp/main"
		if err := df.preprocess(nil); err != nil {
			t.Fatalf("preprocess error %v", err)
		}

		if !contains(df.Build.ArgsBin, runtimeArg) {
			t.Fatalf("missing expected runtime arg: %s", runtimeArg)
		}
	})
}

func TestReadConfigWithWrongPath(t *testing.T) {
	c, err := readConfig("xxxx")
	if err == nil {
		t.Fatal("need throw a error")
	}
	if c != nil {
		t.Fatal("expect is nil but got a conf")
	}
}

func TestKillDelay(t *testing.T) {
	config := Config{
		Build: cfgBuild{
			KillDelay: 1000,
		},
	}
	if config.killDelay() != (1000 * time.Millisecond) {
		t.Fatal("expect KillDelay 1000 to be interpreted as 1000 milliseconds, got ", config.killDelay())
	}
	config.Build.KillDelay = 1
	if config.killDelay() != (1 * time.Millisecond) {
		t.Fatal("expect KillDelay 1 to be interpreted as 1 millisecond, got ", config.killDelay())
	}
	config.Build.KillDelay = 1_000_000
	if config.killDelay() != (1 * time.Millisecond) {
		t.Fatal("expect KillDelay 1_000_000 to be interpreted as 1 millisecond, got ", config.killDelay())
	}
	config.Build.KillDelay = 100_000_000
	if config.killDelay() != (100 * time.Millisecond) {
		t.Fatal("expect KillDelay 100_000_000 to be interpreted as 100 milliseconds, got ", config.killDelay())
	}
	config.Build.KillDelay = 0
	if config.killDelay() != 0 {
		t.Fatal("expect KillDelay 0 to be interpreted as 0, got ", config.killDelay())
	}
}

func contains(sl []string, target string) bool {
	for _, c := range sl {
		if c == target {
			return true
		}
	}
	return false
}

func TestWarnDeprecatedBin(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".air.toml")
	cfgContent := `
[build]
bin = "./tmp/main"
cmd = "go build -o ./tmp/main ."
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	_, _ = InitConfig(cfgPath, nil)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "build.bin is deprecated") {
		t.Fatalf("missing bin deprecation warning in output: %q", output)
	}
}
