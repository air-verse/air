package runner

import (
	"flag"
	"os"
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
	}, {
		name: "Conf",
		path: "_testdata/conf",
		root: "conf_root",
	}, {
		name: "Both",
		path: "_testdata/both",
		root: "both_root",
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

func TestConfPreprocess(t *testing.T) {
	t.Setenv(airWd, "_testdata/toml")
	df := defaultConfig()
	err := df.preprocess(nil)
	if err != nil {
		t.Fatalf("preprocess error %v", err)
	}
	suffix := "/_testdata/toml/tmp/main"
	binPath := df.Build.Bin
	if !strings.HasSuffix(binPath, suffix) {
		t.Fatalf("bin path is %s, but not have suffix  %s.", binPath, suffix)
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

func TestMigrateBinArgs(t *testing.T) {
	tests := []struct {
		name            string
		inputBin        string
		inputArgsBin    []string
		expectedBin     string
		expectedArgsBin []string
		shouldWarn      bool
	}{
		{
			name:            "bin with arguments",
			inputBin:        "./tmp/main server :8080",
			inputArgsBin:    []string{},
			expectedBin:     "./tmp/main",
			expectedArgsBin: []string{"server", ":8080"},
			shouldWarn:      true,
		},
		{
			name:            "bin with arguments and existing args_bin",
			inputBin:        "./tmp/main server",
			inputArgsBin:    []string{":8080"},
			expectedBin:     "./tmp/main",
			expectedArgsBin: []string{"server", ":8080"},
			shouldWarn:      true,
		},
		{
			name:            "bin without arguments",
			inputBin:        "./tmp/main",
			inputArgsBin:    []string{"server", ":8080"},
			expectedBin:     "./tmp/main",
			expectedArgsBin: []string{"server", ":8080"},
			shouldWarn:      false,
		},
		{
			name:            "bin with quoted path containing spaces",
			inputBin:        `"/path/with space/main" arg1 arg2`,
			inputArgsBin:    []string{},
			expectedBin:     `/path/with space/main`,
			expectedArgsBin: []string{"arg1", "arg2"},
			shouldWarn:      true,
		},
		{
			name:            "absolute path with spaces in arguments",
			inputBin:        "/home/user/tmp/main server :8080",
			inputArgsBin:    []string{},
			expectedBin:     "/home/user/tmp/main",
			expectedArgsBin: []string{"server", ":8080"},
			shouldWarn:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Build: cfgBuild{
					Bin:     tt.inputBin,
					ArgsBin: tt.inputArgsBin,
				},
				Log: cfgLog{
					Silent: true, // Silence warnings during test
				},
			}

			config.migrateBinArgs()

			if config.Build.Bin != tt.expectedBin {
				t.Errorf("expected bin='%s', got bin='%s'", tt.expectedBin, config.Build.Bin)
			}

			if len(config.Build.ArgsBin) != len(tt.expectedArgsBin) {
				t.Errorf("expected args_bin length %d, got %d", len(tt.expectedArgsBin), len(config.Build.ArgsBin))
			}

			for i, expected := range tt.expectedArgsBin {
				if i >= len(config.Build.ArgsBin) || config.Build.ArgsBin[i] != expected {
					t.Errorf("expected args_bin[%d]='%s', got '%s'", i, expected, config.Build.ArgsBin[i])
				}
			}
		})
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
