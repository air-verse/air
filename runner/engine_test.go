package runner

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/pelletier/go-toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEngine(t *testing.T) {
	_ = os.Unsetenv(airWd)
	engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	nestedTmpDir := filepath.Join(t.TempDir(), "nested", "build")
	engine.config.TmpDir = nestedTmpDir

	err = engine.checkRunEnv()
	require.NoError(t, err)
	assert.DirExists(t, nestedTmpDir)
}

func TestWatching(t *testing.T) {
	engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.ExcludeRegex = []string{"foo\\.html$", "bar", "_test\\.go"}
	err = engine.config.preprocess(nil)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

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
	engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
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

	engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
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
	err = waitingPortReady(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	t.Logf("port is ready")
	// stop engine
	engine.Stop()
	t.Logf("engine stopped")
	// Wait for engine to fully stop
	err = waitForEngineState(t, engine, false, time.Second*3)
	if err != nil {
		t.Fatalf("engine did not stop: %s.", err)
	}
	wg.Wait()
	assert.True(t, checkPortConnectionRefused(port))
}

func waitingPortConnectionRefused(t *testing.T, port int, timeout time.Duration) error {
	t.Helper()
	t.Logf("waiting port %d connection refused", port)

	// Use environment-aware timeout for CI compatibility
	timeoutMultiplier := 1.0
	if os.Getenv("CI") != "" {
		timeoutMultiplier = 2.0
	}
	adjustedTimeout := time.Duration(float64(timeout) * timeoutMultiplier)

	deadline := time.Now().Add(adjustedTimeout)
	ticker := time.NewTicker(20 * time.Millisecond) // Reduced from 100ms to 20ms
	defer ticker.Stop()

	for {
		_, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if errors.Is(err, syscall.ECONNREFUSED) {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for port %d connection refused (timeout: %v)", port, adjustedTimeout)
		}

		<-ticker.C
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
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.KillDelay = c.Build.KillDelay
	engine.config.Build.Delay = 2000
	engine.config.Build.SendInterrupt = true
	if err := engine.config.preprocess(nil); err != nil {
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
	// Wait for engine to fully stop - the test has kill_delay="2s"
	err = waitForEngineState(t, engine, false, time.Second*5)
	if err != nil {
		t.Logf("engine may not have stopped in time: %s", err)
	}
	assert.False(t, engine.running.Load())
}

func TestCtrlCWhenREngineIsRunning(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)

	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", nil, true)
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
	assert.False(t, engine.running.Load())
}

func TestCtrlCWithFailedBin(t *testing.T) {
	timeout := 5 * time.Second
	done := make(chan struct{})
	go func() {
		dir := initWithQuickExitGoCode(t)
		chdir(t, dir)
		engine, err := NewEngine("", nil, true)
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
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	// Silence engine logs to keep this test output readable.
	engine.config.Log.Silent = true
	silenceBuildCmd(engine.config)
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
	// Wait for first build to fail - reduced from 3s to 500ms
	time.Sleep(time.Millisecond * 500)
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
	if err := waitingPortConnectionRefused(t, port, time.Second*10); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	assert.False(t, engine.running.Load())
}

func TestFixCloseOfChannelAfterTwoFailedBuild(t *testing.T) {
	// fix https://github.com/air-verse/air/issues/294
	// happens after two failed builds
	dir := initWithBuildFailedCode(t)
	// change dir to tmpDir
	chdir(t, dir)
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Log.Silent = true
	silenceBuildCmd(engine.config)
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

	// Wait for first build to complete (with error) - reduced from 3s to 1s
	// Since the build fails immediately, 1s is sufficient
	time.Sleep(time.Millisecond * 500)

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
	// Wait for second build attempt - reduced from 3s to 500ms
	time.Sleep(time.Millisecond * 500)
	// ctrl + c
	sigs <- syscall.SIGINT
	// Wait for engine to stop
	err = waitForEngineState(t, engine, false, time.Second*3)
	if err != nil {
		t.Logf("engine may not have stopped cleanly: %s", err)
	}
	assert.False(t, engine.running.Load())
}

// waitingPortReady waits until the port is ready to be used.
func waitingPortReady(t *testing.T, port int, timeout time.Duration) error {
	t.Helper()
	t.Logf("waiting port %d ready", port)

	// Use environment-aware timeout for CI compatibility
	timeoutMultiplier := 1.0
	if os.Getenv("CI") != "" {
		timeoutMultiplier = 2.0
	}
	adjustedTimeout := time.Duration(float64(timeout) * timeoutMultiplier)

	deadline := time.Now().Add(adjustedTimeout)
	ticker := time.NewTicker(20 * time.Millisecond) // Reduced from 100ms to 20ms
	defer ticker.Stop()

	for {
		conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
		if err == nil {
			_ = conn.Close()
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for port %d ready (timeout: %v)", port, adjustedTimeout)
		}

		<-ticker.C
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
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	go func() {
		engine.Run()
	}()

	// Wait for port to be ready instead of fixed sleep
	err = waitingPortReady(t, port, time.Second*10)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	assert.True(t, checkPortHaveBeenUsed(port))
	t.Logf("try to stop")
	engine.Stop()

	// Wait for engine to stop instead of fixed sleep
	err = waitForEngineState(t, engine, false, time.Second*3)
	if err != nil {
		t.Fatalf("engine did not stop: %s.", err)
	}
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

func silenceBuildCmd(cfg *Config) {
	if cfg == nil {
		return
	}
	if runtime.GOOS == "windows" {
		cfg.Build.Cmd = fmt.Sprintf("%s >NUL 2>&1", cfg.Build.Cmd)
		return
	}
	cfg.Build.Cmd = fmt.Sprintf("%s >/dev/null 2>&1", cfg.Build.Cmd)
}

func TestRebuildWhenRunCmdUsingDLV(t *testing.T) {
	// generate a random port
	port, f := GetPort()
	f()
	t.Logf("port: %d", port)
	tmpDir := initTestEnv(t, port)
	// change dir to tmpDir
	chdir(t, tmpDir)
	engine, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	engine.config.Build.Cmd = "go build -gcflags='all=-N -l' -o ./tmp/main ."
	engine.config.Build.Bin = ""
	dlvPort, f := GetPort()
	f()
	engine.config.Build.FullBin = fmt.Sprintf("dlv exec --accept-multiclient --log --headless --continue --listen :%d --api-version 2 ./tmp/main", dlvPort)
	_ = engine.config.preprocess(nil)
	go func() {
		engine.Run()
	}()
	if err := waitingPortReady(t, port, time.Second*40); err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}

	t.Logf("start change main.go")
	// change file of main.go
	// just append a new empty line to main.go
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
	err = waitingPortReady(t, port, time.Second*40)
	if err != nil {
		t.Fatalf("Should not be fail: %s.", err)
	}
	t.Logf("port is ready")
	// stop engine
	engine.Stop()
	// Wait for engine to stop
	err = waitForEngineState(t, engine, false, time.Second*5)
	if err != nil {
		t.Fatalf("engine did not stop: %s.", err)
	}
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
	// check the file exists
	if _, err := os.Stat(configName); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(configName)
	if err != nil {
		t.Fatal(err)
	}
	expectedPrefix := schemaHeader + "\n\n"
	assert.True(t, strings.HasPrefix(string(raw), expectedPrefix), "config should start with schema header")

	// check the file content is right
	actual, err := readConfig(configName)
	if err != nil {
		t.Fatal(err)
	}
	expect := defaultConfig()
	if len(expect.Build.Entrypoint) == 0 && expect.Build.Bin != "" {
		expect.Build.Entrypoint = entrypoint{expect.Build.Bin}
	}

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
	engine, err := NewEngine(".air.toml", nil, true)
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
	// check the file exists
	if _, err := os.Stat(dftTOML); err != nil {
		t.Fatal(err)
	}
	// check is MacOS
	var cmd *exec.Cmd
	toolName := "sed"

	if runtime.GOOS == "darwin" {
		toolName = "gsed"
	}

	cmd = exec.Command(toolName, "-i", "s/\"_test.*go\"//g", ".air.toml")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Skipf("unable to run %s, make sure the tool is installed to run this test", toolName)
	}

	time.Sleep(time.Second * 2)
	engine, err := NewEngine(".air.toml", nil, false)
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
	engine, err := NewEngine("", nil, true)
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

	engine, err := NewEngine(dftTOML, nil, false)
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

func TestShouldIncludeIncludedFileWithoutIncludedExt(t *testing.T) {
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

	engine, err := NewEngine(dftTOML, nil, false)
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
		err = os.WriteFile("main.sh", []byte("#!/bin/sh\nprintf modified > output"), 0o755)
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

type testExiter struct {
	t          *testing.T
	called     bool
	expectCode int
}

func (te *testExiter) Exit(code int) {
	te.called = true
	if code != te.expectCode {
		te.t.Fatalf("expected exit code %d, got %d", te.expectCode, code)
	}
}

func TestEngineExit(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Engine, chan<- int)
		expectCode int
		wantCalled bool
	}{
		{
			name: "normal exit - no error",
			setup: func(_ *Engine, exitCode chan<- int) {
				go func() {
					exitCode <- 0
				}()
			},
			expectCode: 0,
			wantCalled: false,
		},
		{
			name: "error exit - non-zero code",
			setup: func(_ *Engine, exitCode chan<- int) {
				go func() {
					exitCode <- 1
				}()
			},
			expectCode: 1,
			wantCalled: true,
		},
		{
			name: "process timeout",
			setup: func(_ *Engine, _ chan<- int) {
			},
			expectCode: 0,
			wantCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewEngine("", nil, true)
			if err != nil {
				t.Fatal(err)
			}

			exiter := &testExiter{
				t:          t,
				expectCode: tt.expectCode,
			}
			e.exiter = exiter

			exitCode := make(chan int)

			if tt.setup != nil {
				tt.setup(e, exitCode)
			}
			select {
			case ret := <-exitCode:
				if ret != 0 {
					e.exiter.Exit(ret)
				}
			case <-time.After(1 * time.Millisecond):
				// timeout case
			}

			if tt.wantCalled != exiter.called {
				t.Errorf("Exit() called = %v, want %v", exiter.called, tt.wantCalled)
			}
		})
	}
}

// TestBuildRunRaceCondition tests that a new build does not receive
// stop signals meant for a previous build. This is a regression test for issue #784.
//
// The fix uses a channel-of-channels pattern where each build gets its own unique
// stop channel. When a new build is triggered, it retrieves the previous build's
// stop channel and closes it to signal cancellation.
func TestBuildRunRaceCondition(t *testing.T) {
	e, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	e.config.Log.Silent = true

	// Simulate the race condition scenario from issue #784:
	// 1. Build A starts and puts its stop channel in buildRunCh
	// 2. Build B is triggered, retrieves Build A's channel and closes it
	// 3. Build B puts its own fresh channel in buildRunCh
	// 4. Build B should NOT be affected by Build A's closed channel

	// Simulate Build A putting its stop channel in buildRunCh
	buildAStopCh := make(chan struct{})
	e.buildRunCh <- buildAStopCh

	// Simulate Build B being triggered (mimics what start() does)
	var retrievedChannel chan struct{}
	select {
	case retrievedChannel = <-e.buildRunCh:
		close(retrievedChannel) // Signal Build A to stop
	default:
		t.Fatal("Expected Build A's stop channel to be in buildRunCh")
	}

	// Verify we got Build A's channel
	if retrievedChannel != buildAStopCh {
		t.Error("Should have retrieved Build A's stop channel")
	}

	// Verify Build A's channel is closed
	select {
	case <-buildAStopCh:
		// Good - Build A was signaled to stop
	default:
		t.Error("Build A's stop channel should have been closed")
	}

	// Now simulate Build B starting with its own channel
	buildBStopCh := make(chan struct{})
	e.buildRunCh <- buildBStopCh

	// Build B should NOT be affected by Build A's closed channel
	select {
	case <-buildBStopCh:
		t.Error("Build B's stop channel should NOT be closed yet")
	case <-time.After(50 * time.Millisecond):
		// Good - Build B is still running
	}

	// Test that closing Build B's channel does signal Build B to stop
	close(buildBStopCh)
	select {
	case <-buildBStopCh:
		// Good - Build B received the stop signal
	case <-time.After(50 * time.Millisecond):
		t.Error("Build B should have been stopped when its channel was closed")
	}

	// Clean up - remove Build B's channel from buildRunCh
	select {
	case <-e.buildRunCh:
		// Successfully cleaned up
	default:
		t.Error("Expected Build B's channel to still be in buildRunCh")
	}
}

// TestBuildRunRaceConditionRapidChanges tests rapid file changes don't cause deadlock
func TestBuildRunRaceConditionRapidChanges(t *testing.T) {
	e, err := NewEngine("", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	e.config.Log.Silent = true

	// Simulate 5 rapid builds in succession
	channels := make([]chan struct{}, 5)

	for i := 0; i < 5; i++ {
		// If there's a previous build, stop it
		select {
		case oldCh := <-e.buildRunCh:
			close(oldCh)
		default:
		}

		// Start new build
		channels[i] = make(chan struct{})
		e.buildRunCh <- channels[i]
	}

	// All previous builds should be signaled to stop
	for i := 0; i < 4; i++ {
		select {
		case <-channels[i]:
			// Good - was signaled to stop
		default:
			t.Errorf("Build %d should have been signaled to stop", i)
		}
	}

	// Last build should NOT be stopped
	select {
	case <-channels[4]:
		t.Error("Last build should still be running")
	default:
		// Good
	}

	// Clean up
	<-e.buildRunCh
}

func TestEngineLoadEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	originalValue := "original_global_value"
	t.Setenv("TEST_GLOBAL_VAR", originalValue)

	const initialEnv = `TEST_VAR1=value1
TEST_VAR2=value2
TEST_GLOBAL_VAR=overridden_value
`

	err := os.WriteFile(envPath, []byte(initialEnv), 0o644)
	require.NoError(t, err)

	cfg := defaultConfig()
	cfg.Root = tmpDir
	cfg.EnvFile = ".env"

	engine, err := NewEngineWithConfig(&cfg, false)
	require.NoError(t, err)

	// First load
	engine.loadEnvFile()

	assert.Equal(t, "value1", os.Getenv("TEST_VAR1"), "TEST_VAR1 should be set")
	assert.Equal(t, "value2", os.Getenv("TEST_VAR2"), "TEST_VAR2 should be set")
	assert.Equal(t, "overridden_value", os.Getenv("TEST_GLOBAL_VAR"), "TEST_GLOBAL_VAR should be overridden")

	const updatedEnv = `TEST_VAR1=updated_value1
TEST_GLOBAL_VAR=still_overridden
`
	// Update .env file - change value and remove TEST_VAR2
	err = os.WriteFile(envPath, []byte(updatedEnv), 0o644)
	require.NoError(t, err)

	// Reload
	engine.loadEnvFile()

	assert.Equal(t, "updated_value1", os.Getenv("TEST_VAR1"), "TEST_VAR1 should be updated")
	_, exists := os.LookupEnv("TEST_VAR2")
	assert.False(t, exists, "TEST_VAR2 should be unset after removal from .env")
	assert.Equal(t, "still_overridden", os.Getenv("TEST_GLOBAL_VAR"), "TEST_GLOBAL_VAR should still be overridden")

	// Remove TEST_GLOBAL_VAR from .env - should restore original value
	const finalEnv = `TEST_VAR1=final_value`
	err = os.WriteFile(envPath, []byte(finalEnv), 0o644)
	require.NoError(t, err)

	// Reload again
	engine.loadEnvFile()

	assert.Equal(t, "final_value", os.Getenv("TEST_VAR1"), "TEST_VAR1 should be final")
	assert.Equal(t, originalValue, os.Getenv("TEST_GLOBAL_VAR"), "TEST_GLOBAL_VAR should be restored to original value")
}
