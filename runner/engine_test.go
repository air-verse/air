package runner

import (
	"os"
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	_ = os.Unsetenv(airWd)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if engine.logger == nil {
		t.Fatal("logger should not be nil")
	}
	if engine.config == nil {
		t.Fatal("config should not be nil")
	}
	if engine.watcher == nil {
		t.Fatal("watcher should not be nil")
	}
}

func TestCheckRunEnv(t *testing.T) {
	_ = os.Unsetenv(airWd)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	err = engine.checkRunEnv()
	if err == nil {
		t.Fatal("should throw a err")
	}
}

func TestWatching(t *testing.T) {
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	path, err := os.Getwd()
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	path = strings.Replace(path, "_testdata/toml", "", 1)
	err = engine.watching(path + "/_testdata/watching")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
}

func TestRegexes(t *testing.T) {
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.ExcludeRegex = []string{"foo.html$", "bar"}

	result, err := engine.isExcludeRegex("./test/foo.html")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if result != true {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}

	result, err = engine.isExcludeRegex("./test/bar/index.html")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if result != true {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}

	result, err = engine.isExcludeRegex("./test/unrelated.html")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if result {
		t.Errorf("expected '%t' but got '%t'", false, result)
	}
}

func TestRunBin(t *testing.T) {
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	err = engine.runBin()
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
}

func TestBuildRun(t *testing.T) {
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	engine.buildRun()
}
