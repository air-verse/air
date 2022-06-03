package runner

import (
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func GetPort() (int, func()) {
	l, err := net.Listen("tcp", ":0")
	port := l.Addr().(*net.TCPAddr).Port
	if err != nil {
		panic(err)
	}
	return port, func() {
		_ = l.Close()
	}
}

func TestRun(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	err := os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	go func() {
		engine.Run()
	}()
	time.Sleep(time.Second * 2)
	assert.True(t, checkPortHaveBeenUsed(port))
	t.Logf("try to stop")
	engine.Stop()
	time.Sleep(time.Second * 1)
	assert.False(t, checkPortHaveBeenUsed(port))
	t.Logf("stoped")
}

func checkPortHaveBeenUsed(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func initTestEnv(t *testing.T, port int) string {
	tempDir := t.TempDir()
	t.Logf("tempDir: %s", tempDir)
	// generate golang code to tempdir
	err := generateGoCode(tempDir, port)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	return tempDir
}

// generateGoCode generates golang code to tempdir
func generateGoCode(dir string, port int) error {

	code := fmt.Sprintf(`package main

import (
	"log"
	"net/http"
)

func main() {
	log.Fatal(http.ListenAndServe(":%v", nil))
}
`, port)
	file, err := os.Create(dir + "/main.go")
	if err != nil {
		return err
	}
	_, err = file.WriteString(code)

	// generate go mod file
	mod := `module air.sample.com

go 1.17
`
	file, err = os.Create(dir + "/go.mod")
	if err != nil {
		return err
	}
	_, err = file.WriteString(mod)
	if err != nil {
		return err
	}
	return nil
}
