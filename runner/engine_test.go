package runner

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pelletier/go-toml"
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
		t.Fatal("Config should not be nil")
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
	engine.config.Build.ExcludeRegex = []string{"foo\\.html$", "bar", "_test\\.go"}

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

	result, err = engine.isExcludeRegex("./myPackage/goFile_testxgo")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if result {
		t.Errorf("expected '%t' but got '%t'", false, result)
	}
	result, err = engine.isExcludeRegex("./myPackage/goFile_test.go")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if result != true {
		t.Errorf("expected '%t' but got '%t'", true, result)
	}
}

func TestRunCommand(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)
	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	err = engine.runCommand("touch test.txt")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if _, err := os.Stat("./test.txt"); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Should not be fail: %s.", err)
		}
	}
}

func TestRunPreCmd(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)
	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.PreCmd = []string{"echo 'hello air' > pre_cmd.txt"}
	err = engine.runPreCmd()
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	if _, err := os.Stat("./pre_cmd.txt"); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Should not be fail: %s.", err)
		}
	}
}

func TestRunPostCmd(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)
	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)

	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	engine.config.Build.PostCmd = []string{"echo 'hello air' > post_cmd.txt"}
	err = engine.runPostCmd()
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	if _, err := os.Stat("./post_cmd.txt"); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Should not be fail: %s.", err)
		}
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

func TestRebuild(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	engine.config.Build.ExcludeUnchanged = true
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		engine.Run()
		t.Logf("engine stopped")
		wg.Done()
	}()
	err = waitingPortReady(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	t.Logf("port is ready")

	// start rebuild

	t.Logf("start change main.go")
	// change file of main.go
	// just append a new empty line to main.go
	time.Sleep(time.Second * 2)
	file, err := os.OpenFile("main.go", os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	defer file.Close()
	_, err = file.WriteString("\n")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	err = waitingPortConnectionRefused(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("timeout: %s.", err)
	}
	t.Logf("connection refused")
	time.Sleep(time.Second * 2)
	err = waitingPortReady(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	t.Logf("port is ready")
	// stop engine
	engine.Stop()
	t.Logf("engine stopped")
	wg.Wait()
	time.Sleep(time.Second * 1)
	assert.True(t, checkPortConnectionRefused(port))
}

func waitingPortConnectionRefused(t *testing.T, port int, timeout time.Duration) error {
	t.Logf("waiting port %d connection refused", port)
	timer := time.NewTimer(timeout)
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			return fmt.Errorf("timeout")
		case <-ticker.C:
			print(".")
			_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if errors.Is(err, syscall.ECONNREFUSED) {
				return nil
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}

func TestCtrlCWhenHaveKillDelay(t *testing.T) {
	// fix https://github.com/air-verse/air/issues/278
	// generate a random port
	data := []byte("[build]\n  kill_delay = \"2s\"")
	c := Config{}
	if err := toml.Unmarshal(data, &c); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.KillDelay = c.Build.KillDelay
	engine.config.Build.Delay = 2000
	engine.config.Build.SendInterrupt = true
	if err := engine.config.preprocess(); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	go func() {
		engine.Run()
		t.Logf("engine stopped")
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		<-sigs
		engine.Stop()
		t.Logf("engine stopped")
	}()
	if err := waitingPortReady(t, port, time.Second*10); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	sigs <- syscall.SIGINT
	err = waitingPortConnectionRefused(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	time.Sleep(time.Second * 3)
	assert.False(t, engine.running)
}

func TestCtrlCWhenREngineIsRunning(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	go func() {
		engine.Run()
		t.Logf("engine stopped")
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		engine.Stop()
		t.Logf("engine stopped")
	}()
	if err := waitingPortReady(t, port, time.Second*10); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	sigs <- syscall.SIGINT
	time.Sleep(time.Second * 1)
	err = waitingPortConnectionRefused(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	assert.False(t, engine.running)
}

func TestCtrlCWithFailedBin(t *testing.T) {
	timeout := 5 * time.Second
	done := make(chan struct{})
	go func() {
		dir := initWithQuickExitGoCode(t)
		chdir(t, dir)
		engine, err := NewEngine("", true)
		assert.NoError(t, err)
		engine.config.Build.Bin = "<WRONGCOMAMND>"
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			engine.Run()
			t.Logf("engine stopped")
			wg.Done()
		}()
		go func() {
			<-sigs
			engine.Stop()
			t.Logf("engine stopped")
		}()
		time.Sleep(time.Second * 1)
		sigs <- syscall.SIGINT
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Error("Test timed out")
	}
}

func TestFixCloseOfChannelAfterCtrlC(t *testing.T) {
	// fix https://github.com/air-verse/air/issues/294
	dir := initWithBuildFailedCode(t)
	chdir(t, dir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		engine.Run()
		t.Logf("engine stopped")
	}()

	go func() {
		<-sigs
		engine.Stop()
		t.Logf("engine stopped")
	}()
	// waiting for compile error
	time.Sleep(time.Second * 3)
	port, f := GetPort()
	f()
	// correct code
	err = generateGoCode(dir, port)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	if err := waitingPortReady(t, port, time.Second*10); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	// ctrl + c
	sigs <- syscall.SIGINT
	time.Sleep(time.Second * 1)
	if err := waitingPortConnectionRefused(t, port, time.Second*10); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	assert.False(t, engine.running)
}

func TestFixCloseOfChannelAfterTwoFailedBuild(t *testing.T) {
	// fix https://github.com/air-verse/air/issues/294
	// happens after two failed builds
	dir := initWithBuildFailedCode(t)
	// change dir to tmpDir
	chdir(t, dir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		engine.Run()
		t.Logf("engine stopped")
	}()

	go func() {
		<-sigs
		engine.Stop()
		t.Logf("engine stopped")
	}()

	// waiting for compile error
	time.Sleep(time.Second * 3)

	// edit *.go file to create build error again
	file, err := os.OpenFile("main.go", os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	defer file.Close()
	_, err = file.WriteString("\n")
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	time.Sleep(time.Second * 3)
	// ctrl + c
	sigs <- syscall.SIGINT
	time.Sleep(time.Second * 1)
	assert.False(t, engine.running)
}

// waitingPortReady waits until the port is ready to be used.
func waitingPortReady(t *testing.T, port int, timeout time.Duration) error {
	t.Logf("waiting port %d ready", port)
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(time.Millisecond * 100)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutChan:
			return fmt.Errorf("timeout")
		case <-ticker.C:
			conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}

func TestRun(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
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
	t.Logf("stopped")
}

func checkPortConnectionRefused(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()
	return errors.Is(err, syscall.ECONNREFUSED)
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

func initWithBuildFailedCode(t *testing.T) string {
	tempDir := t.TempDir()
	t.Logf("tempDir: %s", tempDir)
	// generate golang code to tempdir
	err := generateBuildErrorGoCode(tempDir)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	return tempDir
}

func initWithQuickExitGoCode(t *testing.T) string {
	tempDir := t.TempDir()
	t.Logf("tempDir: %s", tempDir)
	// generate golang code to tempdir
	err := generateQuickExitGoCode(tempDir)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	return tempDir
}

func generateQuickExitGoCode(dir string) error {
	code := `package main
// You can edit this code!
// Click here and start typing.

import "fmt"

func main() {
	fmt.Println("Hello, 世界")
}
`
	file, err := os.Create(dir + "/main.go")
	if err != nil {
		return err
	}
	_, err = file.WriteString(code)
	if err != nil {
		return err
	}

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

func generateBuildErrorGoCode(dir string) error {
	code := `package main
// You can edit this code!
// Click here and start typing.

func main() {
	Println("Hello, 世界")

}
`
	file, err := os.Create(dir + "/main.go")
	if err != nil {
		return err
	}
	_, err = file.WriteString(code)
	if err != nil {
		return err
	}

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
	if err != nil {
		return err
	}

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

func TestRebuildWhenRunCmdUsingDLV(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)
	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.Cmd = "go build -gcflags='all=-N -l' -o ./tmp/main ."
	engine.config.Build.Bin = ""
	dlvPort, f := GetPort()
	f()
	engine.config.Build.FullBin = fmt.Sprintf("dlv exec --accept-multiclient --log --headless --continue --listen :%d --api-version 2 ./tmp/main", dlvPort)
	_ = engine.config.preprocess()
	go func() {
		engine.Run()
	}()
	if err := waitingPortReady(t, port, time.Second*40); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	t.Logf("start change main.go")
	// change file of main.go
	// just append a new empty line to main.go
	time.Sleep(time.Second * 2)
	go func() {
		file, err := os.OpenFile("main.go", os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatalf("Should not be fail: %s.", err)
		}
		defer file.Close()
		_, err = file.WriteString("\n")
		if err != nil {
			log.Fatalf("Should not be fail: %s.", err)
		}
	}()
	err = waitingPortConnectionRefused(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("timeout: %s.", err)
	}
	t.Logf("connection refused")
	time.Sleep(time.Second * 2)
	err = waitingPortReady(t, port, time.Second*40)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	t.Logf("port is ready")
	// stop engine
	engine.Stop()
	time.Sleep(time.Second * 3)
	t.Logf("engine stopped")
	assert.True(t, checkPortConnectionRefused(port))
}

func TestWriteDefaultConfig(t *testing.T) {
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	configName, err := writeDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}
	// check the file is exist
	if _, err := os.Stat(configName); err != nil {
		t.Fatal(err)
	}

	// check the file content is right
	actual, err := readConfig(configName)
	if err != nil {
		t.Fatal(err)
	}
	expect := defaultConfig()

	assert.Equal(t, expect, *actual)
}

func TestCheckNilSliceShouldBeenOverwrite(t *testing.T) {
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)

	// change dir to tmpDir
	chdir(t, tmpDir)

	// write easy config file

	config := `
[build]
cmd = "go build ."
bin = "tmp/main"
exclude_regex = []
exclude_dir = ["test"]
exclude_file = ["main.go"]
include_file = ["test/not_a_test.go"]

`
	if err := os.WriteFile(dftTOML, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(".air.toml", true)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"go", "tpl", "tmpl", "html"}, engine.config.Build.IncludeExt)
	assert.Equal(t, []string{}, engine.config.Build.ExcludeRegex)
	assert.Equal(t, []string{"test"}, engine.config.Build.ExcludeDir)
	// add new config
	assert.Equal(t, []string{"main.go"}, engine.config.Build.ExcludeFile)
	assert.Equal(t, []string{"test/not_a_test.go"}, engine.config.Build.IncludeFile)
	assert.Equal(t, "go build .", engine.config.Build.Cmd)
}

func TestShouldIncludeGoTestFile(t *testing.T) {
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	_, err := writeDefaultConfig()
	if err != nil {
		t.Fatal(err)
	}

	// write go test file
	file, err := os.Create("main_test.go")
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString(`package main

import "testing"

func Test(t *testing.T) {
	t.Log("testing")
}
`)
	if err != nil {
		t.Fatal(err)
	}
	// run sed
	// check the file is exist
	if _, err := os.Stat(dftTOML); err != nil {
		t.Fatal(err)
	}
	// check is MacOS
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("gsed", "-i", "s/\"_test.*go\"//g", ".air.toml")
	} else {
		cmd = exec.Command("sed", "-i", "s/\"_test.*go\"//g", ".air.toml")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second * 2)
	engine, err := NewEngine(".air.toml", false)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		engine.Run()
	}()

	t.Logf("start change main_test.go")
	// change file of main_test.go
	// just append a new empty line to main_test.go
	if err = waitingPortReady(t, port, time.Second*40); err != nil {
		t.Fatal(err)
	}
	go func() {
		file, err = os.OpenFile("main_test.go", os.O_APPEND|os.O_WRONLY, 0o644)
		assert.NoError(t, err)
		defer file.Close()
		_, err = file.WriteString("\n")
		assert.NoError(t, err)
	}()
	// should Have rebuild
	if err = waitingPortReady(t, port, time.Second*10); err != nil {
		t.Fatal(err)
	}
}

func TestCreateNewDir(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	go func() {
		engine.Run()
	}()
	time.Sleep(time.Second * 2)
	assert.True(t, checkPortHaveBeenUsed(port))

	// create a new dir make dir
	if err = os.Mkdir(tmpDir+"/dir", 0o644); err != nil {
		t.Fatal(err)
	}

	// no need reload
	if err = waitingPortConnectionRefused(t, port, 3*time.Second); err == nil {
		t.Fatal("should raise a error")
	}
	engine.Stop()
	time.Sleep(2 * time.Second)
}

func TestShouldIncludeIncludedFile(t *testing.T) {
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)

	chdir(t, tmpDir)

	config := `
[build]
cmd = "true" # do nothing
full_bin = "sh main.sh"
include_ext = ["sh"]
include_dir = ["nonexist"] # prevent default "." watch from taking effect
include_file = ["main.sh"]
`
	if err := os.WriteFile(dftTOML, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	err := os.WriteFile("main.sh", []byte("#!/bin/sh\nprintf original > output"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	engine, err := NewEngine(dftTOML, false)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		engine.Run()
	}()

	time.Sleep(time.Second * 1)

	bytes, err := os.ReadFile("output")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("original"), bytes)

	t.Logf("start change main.sh")
	go func() {
		err := os.WriteFile("main.sh", []byte("#!/bin/sh\nprintf modified > output"), 0o755)
		if err != nil {
			log.Fatalf("Error updating file: %s.", err)
		}
	}()

	time.Sleep(time.Second * 3)

	bytes, err = os.ReadFile("output")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("modified"), bytes)
}

func TestShouldIncludeIncludedFileWihtoutIncludedExt(t *testing.T) {
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)

	chdir(t, tmpDir)

	config := `
[build]
cmd = "true" # do nothing
full_bin = "sh main.sh"
include_ext = ["go"]
include_dir = ["nonexist"] # prevent default "." watch from taking effect
include_file = ["main.sh"]
`
	if err := os.WriteFile(dftTOML, []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	err := os.WriteFile("main.sh", []byte("#!/bin/sh\nprintf original > output"), 0o755)
	if err != nil {
		t.Fatal(err)
	}

	engine, err := NewEngine(dftTOML, false)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		engine.Run()
	}()

	time.Sleep(time.Second * 1)

	bytes, err := os.ReadFile("output")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("original"), bytes)

	t.Logf("start change main.sh")
	go func() {
		err := os.WriteFile("main.sh", []byte("#!/bin/sh\nprintf modified > output"), 0o755)
		if err != nil {
			log.Fatalf("Error updating file: %s.", err)
		}
	}()

	time.Sleep(time.Second * 3)

	bytes, err = os.ReadFile("output")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []byte("modified"), bytes)
}
