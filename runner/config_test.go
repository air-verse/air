package runner

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

const (
	bin = `./tmp/main`
	cmd = "go build -o ./tmp/main ."
)

func getWindowsConfig() Config {
	build := cfgBuild{
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
	err = c.preprocess()
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Setenv(airWd, tt.path)
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
	_ = os.Setenv(airWd, "_testdata/toml")
	df := defaultConfig()
	err := df.preprocess()
	if err != nil {
		t.Fatalf("preprocess error %v", err)
	}
	suffix := "/_testdata/toml/tmp/main"
	binPath := df.Build.Bin
	if !strings.HasSuffix(binPath, suffix) {
		t.Fatalf("bin path is %s, but not have suffix  %s.", binPath, suffix)
	}
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
