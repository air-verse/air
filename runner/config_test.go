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
		CfgBuildCommon: CfgBuildCommon{
			PreCmd: []string{"echo Hello Air"},
			Cmd:    "go build -o ./tmp/main .",
			Bin:    "./tmp/main",
		},
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
	t.Parallel()
	var err error

	c := getWindowsConfig()
	err = c.preprocess(nil)
	if err != nil {
		t.Fatal(err)
	}

	if runtime.GOOS == "windows" {
		if strings.HasSuffix(c.Build.Bin, "exe") {
			t.Fail()
		}

		if strings.Contains(c.Build.Bin, "exe") {
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

func TestDefaultPathConfigWithInvalidTOML(t *testing.T) {
	// Test that defaultPathConfig returns an error when .air.toml exists but has parse errors
	// This is a regression test for issue #678
	t.Setenv(airWd, "_testdata/invalid_toml")
	_, err := defaultPathConfig()
	if err == nil {
		t.Fatal("expected error when .air.toml has parse errors, but got nil")
	}
	if !strings.Contains(err.Error(), "failed to parse") {
		t.Fatalf("expected error message to contain 'failed to parse', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "defined twice") {
		t.Fatalf("expected error message to contain 'defined twice', got: %s", err.Error())
	}
}

func TestConfPreprocess(t *testing.T) {
	t.Setenv(airWd, "_testdata/toml")
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to getwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	df := defaultConfig()
	err = df.preprocess(nil)
	if err != nil {
		t.Fatalf("preprocess error %v", err)
	}
	suffix := filepath.Join("_testdata", "toml", "tmp", "main")
	if runtime.GOOS == "windows" {
		suffix += ".exe"
	}
	binPath := df.Build.Bin
	if !strings.HasSuffix(binPath, suffix) {
		t.Fatalf("bin path is %s, but not have suffix  %s.", binPath, suffix)
	}
}

func TestEntrypointResolvesAbsolutePath(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	c, err := readConfig("xxxx")
	if err == nil {
		t.Fatal("need throw a error")
	}
	if c != nil {
		t.Fatal("expect is nil but got a conf")
	}
}

func TestKillDelay(t *testing.T) {
	t.Parallel()
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

func TestApplyBuildOverrides(t *testing.T) {
	base := defaultConfigBase()
	override := &cfgBuildOverrides{
		CfgBuildCommon: CfgBuildCommon{
			PreCmd:  []string{"echo override"},
			Cmd:     "go build -o ./tmp/custom .",
			ArgsBin: []string{"custom"},
		},
	}

	applyBuildOverrides(&base.Build, override)

	if base.Build.Cmd != override.Cmd {
		t.Fatalf("cmd mismatch: got %s want %s", base.Build.Cmd, override.Cmd)
	}
	if !reflect.DeepEqual(base.Build.PreCmd, override.PreCmd) {
		t.Fatalf("pre_cmd mismatch: got %v want %v", base.Build.PreCmd, override.PreCmd)
	}
	if !reflect.DeepEqual(base.Build.ArgsBin, override.ArgsBin) {
		t.Fatalf("args_bin mismatch: got %v want %v", base.Build.ArgsBin, override.ArgsBin)
	}
	if base.Build.Bin != "./tmp/main" {
		t.Fatalf("bin should remain default, got %s", base.Build.Bin)
	}
}

func TestAddPlatformOverridesForInit(t *testing.T) {
	cfg := defaultConfigBase()
	setEntrypointFromBin(&cfg)
	addPlatformOverridesForInit(&cfg, PlatformWindows)

	if cfg.Build.Windows == nil {
		t.Fatal("expected windows overrides to be set")
	}
	if cfg.Build.Windows.Cmd != "go build -o ./tmp/main.exe ." {
		t.Fatalf("windows cmd mismatch: got %s", cfg.Build.Windows.Cmd)
	}
	if cfg.Build.Windows.Bin != `tmp\main.exe` {
		t.Fatalf("windows bin mismatch: got %s", cfg.Build.Windows.Bin)
	}
	if !reflect.DeepEqual(cfg.Build.Windows.Entrypoint, entrypoint{`tmp\main.exe`}) {
		t.Fatalf("windows entrypoint mismatch: got %v", cfg.Build.Windows.Entrypoint)
	}
}

func TestDefaultConfigForOS(t *testing.T) {
	t.Parallel()

	winCfg := defaultConfigForOS(PlatformWindows)
	if winCfg.Build.Cmd != "go build -o ./tmp/main.exe ." {
		t.Fatalf("windows cmd mismatch: got %q", winCfg.Build.Cmd)
	}
	if winCfg.Build.Bin != `tmp\main.exe` {
		t.Fatalf("windows bin mismatch: got %q", winCfg.Build.Bin)
	}

	linuxCfg := defaultConfigForOS("linux")
	if linuxCfg.Build.Cmd != "go build -o ./tmp/main ." {
		t.Fatalf("linux cmd mismatch: got %q", linuxCfg.Build.Cmd)
	}
	if linuxCfg.Build.Bin != "./tmp/main" {
		t.Fatalf("linux bin mismatch: got %q", linuxCfg.Build.Bin)
	}
	if linuxCfg.Misc.StartupBanner != nil {
		t.Fatalf("startup_banner should default to nil, got %v", *linuxCfg.Misc.StartupBanner)
	}
}

func TestWithArgsSetsStartupBanner(t *testing.T) {
	t.Parallel()

	t.Run("custom text", func(t *testing.T) {
		cfg := defaultConfig()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		args := ParseConfigFlag(fs)
		value := "Watcher A"
		info, ok := args["misc.startup_banner"]
		if !ok {
			t.Fatal("misc.startup_banner flag mapping missing")
		}
		*info.Value = value

		cfg.withArgs(args)

		if cfg.Misc.StartupBanner == nil {
			t.Fatal("startup_banner should be set")
		}
		if got := *cfg.Misc.StartupBanner; got != value {
			t.Fatalf("startup_banner mismatch: got %q want %q", got, value)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		cfg := defaultConfig()
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		args := ParseConfigFlag(fs)
		info, ok := args["misc.startup_banner"]
		if !ok {
			t.Fatal("misc.startup_banner flag mapping missing")
		}
		*info.Value = ""

		cfg.withArgs(args)

		if cfg.Misc.StartupBanner == nil {
			t.Fatal("startup_banner should be set")
		}
		if got := *cfg.Misc.StartupBanner; got != "" {
			t.Fatalf("startup_banner mismatch: got %q want empty", got)
		}
	})
}

func TestInitConfigForDisplayStartupBanner(t *testing.T) {
	t.Parallel()

	t.Run("reads empty string from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, ".air.toml")
		cfgContent := `
[misc]
startup_banner = ""
`
		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		cfg, err := InitConfigForDisplay(cfgPath, nil)
		if err != nil {
			t.Fatalf("InitConfigForDisplay returned error: %v", err)
		}

		if cfg.Misc.StartupBanner == nil {
			t.Fatal("startup_banner should be set")
		}
		if got := *cfg.Misc.StartupBanner; got != "" {
			t.Fatalf("startup_banner mismatch: got %q want empty", got)
		}
	})

	t.Run("applies command arg override", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, ".air.toml")
		cfgContent := `
[misc]
startup_banner = "FromConfig"
`
		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		args := ParseConfigFlag(fs)
		value := "FromFlag"
		info, ok := args["misc.startup_banner"]
		if !ok {
			t.Fatal("misc.startup_banner flag mapping missing")
		}
		*info.Value = value

		cfg, err := InitConfigForDisplay(cfgPath, args)
		if err != nil {
			t.Fatalf("InitConfigForDisplay returned error: %v", err)
		}

		if cfg.Misc.StartupBanner == nil {
			t.Fatal("startup_banner should be set")
		}
		if got := *cfg.Misc.StartupBanner; got != value {
			t.Fatalf("startup_banner mismatch: got %q want %q", got, value)
		}
	})
}

func TestPlatformBuildOverridesSelection(t *testing.T) {
	t.Parallel()

	win := &cfgBuildOverrides{CfgBuildCommon: CfgBuildCommon{Cmd: "win"}}
	darwin := &cfgBuildOverrides{CfgBuildCommon: CfgBuildCommon{Cmd: "darwin"}}
	linux := &cfgBuildOverrides{CfgBuildCommon: CfgBuildCommon{Cmd: "linux"}}
	build := &cfgBuild{Windows: win, Darwin: darwin, Linux: linux}

	if got := platformBuildOverrides(build, PlatformWindows); got != win {
		t.Fatalf("windows override mismatch: got %v", got)
	}
	if got := platformBuildOverrides(build, "darwin"); got != darwin {
		t.Fatalf("darwin override mismatch: got %v", got)
	}
	if got := platformBuildOverrides(build, "linux"); got != linux {
		t.Fatalf("linux override mismatch: got %v", got)
	}
	if got := platformBuildOverrides(build, "freebsd"); got != nil {
		t.Fatalf("unknown platform should return nil, got %v", got)
	}
	if got := platformBuildOverrides(nil, PlatformWindows); got != nil {
		t.Fatalf("nil build should return nil, got %v", got)
	}
}

func TestBuildOverridesFromDiff(t *testing.T) {
	t.Parallel()

	base := defaultConfigBase().Build
	if got := buildOverridesFromDiff(base, base); got != nil {
		t.Fatalf("expected nil override for identical configs, got %v", got)
	}

	target := base
	target.PreCmd = []string{"echo pre"}
	target.Cmd = "go build -o ./tmp/custom ."
	target.PostCmd = []string{"echo post"}
	target.Bin = "./tmp/custom"
	target.Entrypoint = entrypoint{"./tmp/custom", "serve"}
	target.FullBin = "APP_ENV=dev ./tmp/custom"
	target.ArgsBin = []string{"--port", "8080"}

	got := buildOverridesFromDiff(base, target)
	if got == nil {
		t.Fatal("expected non-nil override for changed configs")
	}
	if !reflect.DeepEqual(got.PreCmd, target.PreCmd) {
		t.Fatalf("pre_cmd mismatch: got %v want %v", got.PreCmd, target.PreCmd)
	}
	if got.Cmd != target.Cmd {
		t.Fatalf("cmd mismatch: got %q want %q", got.Cmd, target.Cmd)
	}
	if !reflect.DeepEqual(got.PostCmd, target.PostCmd) {
		t.Fatalf("post_cmd mismatch: got %v want %v", got.PostCmd, target.PostCmd)
	}
	if got.Bin != target.Bin {
		t.Fatalf("bin mismatch: got %q want %q", got.Bin, target.Bin)
	}
	if !reflect.DeepEqual(got.Entrypoint, target.Entrypoint) {
		t.Fatalf("entrypoint mismatch: got %v want %v", got.Entrypoint, target.Entrypoint)
	}
	if got.FullBin != target.FullBin {
		t.Fatalf("full_bin mismatch: got %q want %q", got.FullBin, target.FullBin)
	}
	if !reflect.DeepEqual(got.ArgsBin, target.ArgsBin) {
		t.Fatalf("args_bin mismatch: got %v want %v", got.ArgsBin, target.ArgsBin)
	}
}

func TestSetEntrypointFromBin(t *testing.T) {
	t.Parallel()

	cfg := defaultConfigBase()
	setEntrypointFromBin(&cfg)
	if !reflect.DeepEqual(cfg.Build.Entrypoint, entrypoint{"./tmp/main"}) {
		t.Fatalf("entrypoint mismatch: got %v", cfg.Build.Entrypoint)
	}

	cfgWithEntry := defaultConfigBase()
	cfgWithEntry.Build.Entrypoint = entrypoint{"./tmp/custom"}
	setEntrypointFromBin(&cfgWithEntry)
	if !reflect.DeepEqual(cfgWithEntry.Build.Entrypoint, entrypoint{"./tmp/custom"}) {
		t.Fatalf("existing entrypoint should not be overwritten, got %v", cfgWithEntry.Build.Entrypoint)
	}

	cfgEmptyBin := defaultConfigBase()
	cfgEmptyBin.Build.Bin = ""
	setEntrypointFromBin(&cfgEmptyBin)
	if len(cfgEmptyBin.Build.Entrypoint) != 0 {
		t.Fatalf("entrypoint should remain empty when bin is empty, got %v", cfgEmptyBin.Build.Entrypoint)
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

func TestInitConfigWithoutConfigDoesNotWarnDeprecatedBin(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(airWd, tmpDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to getwd: %v", err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalDir); chdirErr != nil {
			t.Fatalf("failed to restore working directory: %v", chdirErr)
		}
	})

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = oldStderr
	})

	if _, err := InitConfig("", nil); err != nil {
		t.Fatalf("InitConfig returned error: %v", err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	output := string(out)
	if strings.Contains(output, "build.bin is deprecated") {
		t.Fatalf("unexpected bin deprecation warning in output: %q", output)
	}
}

func TestWarnDeprecatedBin(t *testing.T) {
	t.Parallel()
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

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	_, _ = InitConfig(cfgPath, nil)

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stderr = oldStderr

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	output := string(out)
	if !strings.Contains(output, "build.bin is deprecated") {
		t.Fatalf("missing bin deprecation warning in output: %q", output)
	}
}

func TestInitConfigAppliesWindowsBuildOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".air.toml")
	cfgContent := `
[build]
cmd = "base-cmd"
entrypoint = ["./tmp/base"]
args_bin = ["base-arg"]

[build.windows]
cmd = "windows-cmd"
entrypoint = ["tmp\\main.exe"]
args_bin = ["win-arg"]
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := InitConfig(cfgPath, nil)
	if err != nil {
		t.Fatalf("InitConfig error: %v", err)
	}

	if runtime.GOOS == PlatformWindows {
		if cfg.Build.Cmd != "windows-cmd" {
			t.Fatalf("expected windows cmd, got %q", cfg.Build.Cmd)
		}
		if !contains(cfg.Build.ArgsBin, "win-arg") {
			t.Fatalf("expected windows args_bin to contain %q, got %v", "win-arg", cfg.Build.ArgsBin)
		}
		if !strings.HasSuffix(cfg.Build.Entrypoint.binary(), filepath.Join("tmp", "main.exe")) {
			t.Fatalf("expected windows entrypoint suffix, got %q", cfg.Build.Entrypoint.binary())
		}
		return
	}

	if cfg.Build.Cmd != "base-cmd" {
		t.Fatalf("expected base cmd on non-windows, got %q", cfg.Build.Cmd)
	}
	if !contains(cfg.Build.ArgsBin, "base-arg") {
		t.Fatalf("expected base args_bin to contain %q on non-windows, got %v", "base-arg", cfg.Build.ArgsBin)
	}
	if !strings.HasSuffix(cfg.Build.Entrypoint.binary(), filepath.Join("tmp", "base")) {
		t.Fatalf("expected base entrypoint suffix, got %q", cfg.Build.Entrypoint.binary())
	}
}

func TestInitConfigAppliesCurrentPlatformBuildOverride(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".air.toml")
	cfgContent := `
[build]
cmd = "base-cmd"
entrypoint = ["./tmp/base"]
args_bin = ["base-arg"]

[build.windows]
cmd = "windows-cmd"
entrypoint = ["tmp\\main.exe"]
args_bin = ["win-arg"]

[build.darwin]
cmd = "darwin-cmd"
entrypoint = ["./tmp/darwin"]
args_bin = ["darwin-arg"]

[build.linux]
cmd = "linux-cmd"
entrypoint = ["./tmp/linux"]
args_bin = ["linux-arg"]
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := InitConfig(cfgPath, nil)
	if err != nil {
		t.Fatalf("InitConfig error: %v", err)
	}

	switch runtime.GOOS {
	case PlatformWindows:
		if cfg.Build.Cmd != "windows-cmd" {
			t.Fatalf("expected windows cmd, got %q", cfg.Build.Cmd)
		}
		if !contains(cfg.Build.ArgsBin, "win-arg") {
			t.Fatalf("expected windows args_bin to contain %q, got %v", "win-arg", cfg.Build.ArgsBin)
		}
		if !strings.HasSuffix(cfg.Build.Entrypoint.binary(), filepath.Join("tmp", "main.exe")) {
			t.Fatalf("expected windows entrypoint suffix, got %q", cfg.Build.Entrypoint.binary())
		}
	case "darwin":
		if cfg.Build.Cmd != "darwin-cmd" {
			t.Fatalf("expected darwin cmd, got %q", cfg.Build.Cmd)
		}
		if !contains(cfg.Build.ArgsBin, "darwin-arg") {
			t.Fatalf("expected darwin args_bin to contain %q, got %v", "darwin-arg", cfg.Build.ArgsBin)
		}
		if !strings.HasSuffix(cfg.Build.Entrypoint.binary(), filepath.Join("tmp", "darwin")) {
			t.Fatalf("expected darwin entrypoint suffix, got %q", cfg.Build.Entrypoint.binary())
		}
	default:
		if cfg.Build.Cmd != "linux-cmd" {
			t.Fatalf("expected linux cmd, got %q", cfg.Build.Cmd)
		}
		if !contains(cfg.Build.ArgsBin, "linux-arg") {
			t.Fatalf("expected linux args_bin to contain %q, got %v", "linux-arg", cfg.Build.ArgsBin)
		}
		if !strings.HasSuffix(cfg.Build.Entrypoint.binary(), filepath.Join("tmp", "linux")) {
			t.Fatalf("expected linux entrypoint suffix, got %q", cfg.Build.Entrypoint.binary())
		}
	}
}

func TestTmpDirAdjustsDefaults(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".air.toml")
	cfgContent := `tmp_dir = ".tmp"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := InitConfig(cfgPath, nil)
	if err != nil {
		t.Fatalf("InitConfig error: %v", err)
	}

	if !strings.Contains(cfg.Build.Cmd, ".tmp") {
		t.Fatalf("expected Build.Cmd to reference .tmp, got %s", cfg.Build.Cmd)
	}
	if strings.Contains(cfg.Build.Cmd, "./tmp/") {
		t.Fatalf("expected Build.Cmd to not reference ./tmp/, got %s", cfg.Build.Cmd)
	}

	binBase := filepath.Base(cfg.Build.Bin)
	if runtime.GOOS == "windows" {
		if binBase != "main.exe" {
			t.Fatalf("unexpected bin base: %s", binBase)
		}
	} else {
		if binBase != "main" {
			t.Fatalf("unexpected bin base: %s", binBase)
		}
	}
	if !strings.Contains(cfg.Build.Bin, ".tmp") {
		t.Fatalf("expected Build.Bin to reference .tmp, got %s", cfg.Build.Bin)
	}

	foundTmpInExclude := false
	foundDotTmpInExclude := false
	for _, dir := range cfg.Build.ExcludeDir {
		if dir == "tmp" {
			foundTmpInExclude = true
		}
		if dir == ".tmp" {
			foundDotTmpInExclude = true
		}
	}
	if foundTmpInExclude {
		t.Fatal("expected ExcludeDir to not contain 'tmp'")
	}
	if !foundDotTmpInExclude {
		t.Fatal("expected ExcludeDir to contain '.tmp'")
	}
}

func TestTmpDirAdjustsDefaultsWindows(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		TmpDir: ".tmp",
		Build: cfgBuild{
			CfgBuildCommon: CfgBuildCommon{
				Cmd: "go build -o ./tmp/main.exe .",
				Bin: `tmp\main.exe`,
			},
			ExcludeDir: []string{"assets", "tmp", "vendor", "testdata"},
		},
	}

	cfg.adjustDefaultsForTmpDirWithOS("windows")

	expectedCmd := "go build -o ./.tmp/main.exe ."
	if cfg.Build.Cmd != expectedCmd {
		t.Fatalf("expected Build.Cmd %q, got %q", expectedCmd, cfg.Build.Cmd)
	}
	expectedBin := `.tmp\main.exe`
	if cfg.Build.Bin != expectedBin {
		t.Fatalf("expected Build.Bin %q, got %q", expectedBin, cfg.Build.Bin)
	}
	foundDotTmp := false
	for _, dir := range cfg.Build.ExcludeDir {
		if dir == "tmp" {
			t.Fatal("expected ExcludeDir to not contain 'tmp'")
		}
		if dir == ".tmp" {
			foundDotTmp = true
		}
	}
	if !foundDotTmp {
		t.Fatal("expected ExcludeDir to contain '.tmp'")
	}
}

func TestTmpDirDoesNotOverrideExplicitCmd(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, ".air.toml")
	cfgContent := `tmp_dir = ".tmp"

[build]
cmd = "make build"
bin = "./bin/myapp"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := InitConfig(cfgPath, nil)
	if err != nil {
		t.Fatalf("InitConfig error: %v", err)
	}

	if cfg.Build.Cmd != "make build" {
		t.Fatalf("expected Build.Cmd to remain 'make build', got %s", cfg.Build.Cmd)
	}
	if !strings.Contains(cfg.Build.Bin, "myapp") {
		t.Fatalf("expected Build.Bin to contain 'myapp', got %s", cfg.Build.Bin)
	}
}

func TestTmpDirAdjustsDefaultsWithAbsolutePath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == PlatformWindows {
		t.Skip("POSIX absolute path test only runs on linux/macos")
	}

	cfg := &Config{
		TmpDir: "/tmp/air-build",
		Build: cfgBuild{
			CfgBuildCommon: CfgBuildCommon{
				Cmd: "go build -o ./tmp/main .",
				Bin: "./tmp/main",
			},
			ExcludeDir: []string{"assets", "tmp", "vendor", "testdata"},
		},
	}

	cfg.adjustDefaultsForTmpDirWithOS("linux")

	expectedCmd := "go build -o /tmp/air-build/main ."
	if cfg.Build.Cmd != expectedCmd {
		t.Fatalf("expected Build.Cmd %q, got %q", expectedCmd, cfg.Build.Cmd)
	}
	expectedBin := "/tmp/air-build/main"
	if cfg.Build.Bin != expectedBin {
		t.Fatalf("expected Build.Bin %q, got %q", expectedBin, cfg.Build.Bin)
	}
}

func TestTmpDirAdjustsDefaultsWithWindowsAbsolutePath(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != PlatformWindows {
		t.Skip("Windows absolute path test only runs on windows")
	}

	cfg := &Config{
		TmpDir: `C:\tmp\air-build`,
		Build: cfgBuild{
			CfgBuildCommon: CfgBuildCommon{
				Cmd: "go build -o ./tmp/main.exe .",
				Bin: `tmp\main.exe`,
			},
			ExcludeDir: []string{"assets", "tmp", "vendor", "testdata"},
		},
	}

	cfg.adjustDefaultsForTmpDirWithOS("windows")

	expectedCmd := "go build -o C:/tmp/air-build/main.exe ."
	if cfg.Build.Cmd != expectedCmd {
		t.Fatalf("expected Build.Cmd %q, got %q", expectedCmd, cfg.Build.Cmd)
	}
	expectedBin := `C:\tmp\air-build\main.exe`
	if cfg.Build.Bin != expectedBin {
		t.Fatalf("expected Build.Bin %q, got %q", expectedBin, cfg.Build.Bin)
	}
}

func TestTmpDirDoesNotOverrideExplicitExcludeDir(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == PlatformWindows {
		t.Skip("POSIX absolute path test only runs on linux/macos")
	}

	cfg := &Config{
		TmpDir: ".tmp",
		Build: cfgBuild{
			CfgBuildCommon: CfgBuildCommon{
				Cmd: "go build -o ./tmp/main .",
				Bin: "./tmp/main",
			},
			ExcludeDir: []string{"tmp", "node_modules"},
		},
	}

	cfg.adjustDefaultsForTmpDirWithOS("linux")

	if cfg.Build.ExcludeDir[0] != "tmp" {
		t.Fatalf("expected first exclude_dir value to stay 'tmp', got %q", cfg.Build.ExcludeDir[0])
	}
	if cfg.Build.ExcludeDir[1] != "node_modules" {
		t.Fatalf("expected second exclude_dir value to stay 'node_modules', got %q", cfg.Build.ExcludeDir[1])
	}
}

func TestWarnIgnoreDangerousRootDirProtection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("root dir protection uses Unix root path")
	}

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Run("when ignore_dangerous_root_dir is true", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, ".air.toml")
		cfgContent := `
root = "/"

[build]
entrypoint = "tmp/main"
cmd = "go build -o ./tmp/main ."
ignore_dangerous_root_dir = true
`
		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stderr = w

		_, _ = InitConfig(cfgPath, nil)

		if err := w.Close(); err != nil {
			t.Fatalf("failed to close writer: %v", err)
		}
		os.Stderr = oldStderr

		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("failed to read output: %v", err)
		}
		output := string(out)
		if !strings.Contains(output, "ignoring root directory protections. This could cause excessive file watching. It is recommended to run air in a project directory") {
			t.Fatalf("missing root directory protection warning in output: %q", output)
		}
	})
	t.Run("when ignore_dangerous_root_dir is false", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, ".air.toml")
		cfgContent := `
root = "/"

[build]
entrypoint = "tmp/main"
cmd = "go build -o ./tmp/main ."
ignore_dangerous_root_dir = false
`
		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stderr = w

		_, _ = InitConfig(cfgPath, nil)

		if err := w.Close(); err != nil {
			t.Fatalf("failed to close writer: %v", err)
		}
		os.Stderr = oldStderr

		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("failed to read output: %v", err)
		}
		output := string(out)
		if strings.Contains(output, "ignoring root directory protections") {
			t.Fatalf("unexpected root directory protection warning in output: %q", output)
		}
	})

	t.Run("when ignore_dangerous_root_dir is not set", func(t *testing.T) {
		cfgPath := filepath.Join(tmpDir, ".air.toml")
		cfgContent := `
root = "/"

[build]
entrypoint = "tmp/main"
cmd = "go build -o ./tmp/main ."
`
		if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}

		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stderr = w

		_, _ = InitConfig(cfgPath, nil)

		if err := w.Close(); err != nil {
			t.Fatalf("failed to close writer: %v", err)
		}
		os.Stderr = oldStderr

		out, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("failed to read output: %v", err)
		}
		output := string(out)
		if strings.Contains(output, "ignoring root directory protections") {
			t.Fatalf("unexpected root directory protection warning in output: %q", output)
		}
	})
}
