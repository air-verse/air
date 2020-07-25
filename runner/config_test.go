package runner

import (
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
