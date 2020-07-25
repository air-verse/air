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

func getWindowsConfig() config {
	build := cfgBuild{
		Cmd:         "go build -o ./tmp/main .",
		Bin:         "./tmp/main",
		Log:         "build-errors.log",
		IncludeExt:  []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir:  []string{"assets", "tmp", "vendor"},
		Delay:       1000,
		StopOnError: true,
	}
	if runtime.GOOS == "windows" {
		build.Bin = bin
		build.Cmd = cmd
	}

	return config{
		Root:   ".",
		TmpDir: "tmp",
		Build:  build,
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
			os.Setenv(airWd, tt.path)
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
