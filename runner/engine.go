package runner

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gohugoio/hugo/watcher/filenotify"
)

// Engine ...
type Engine struct {
	config *Config

	exiter    exiter
	proxy     *Proxy
	logger    *logger
	watcher   filenotify.FileWatcher
	debugMode bool
	runArgs   []string
	running   atomic.Bool

	eventCh        chan string
	watcherStopCh  chan bool
	buildRunCh     chan bool
	buildRunStopCh chan bool
	// binStopCh is a channel for process termination control
	// Type chan<- chan int indicates it's a send-only channel that transmits another channel(chan int)
	binStopCh chan<- chan int
	exitCh    chan bool

	mu            sync.RWMutex
	watchers      uint
	fileChecksums *checksumMap

	ll sync.Mutex // lock for logger
}

// NewEngineWithConfig ...
func NewEngineWithConfig(cfg *Config, debugMode bool) (*Engine, error) {
	logger := newLogger(cfg)
	watcher, err := newWatcher(cfg)
	if err != nil {
		return nil, err
	}
	e := Engine{
		config:         cfg,
		exiter:         defaultExiter{},
		proxy:          NewProxy(&cfg.Proxy),
		logger:         logger,
		watcher:        watcher,
		debugMode:      debugMode,
		runArgs:        cfg.Build.ArgsBin,
		eventCh:        make(chan string, 1000),
		watcherStopCh:  make(chan bool, 10),
		buildRunCh:     make(chan bool, 1),
		buildRunStopCh: make(chan bool, 1),
		exitCh:         make(chan bool),
		fileChecksums:  &checksumMap{m: make(map[string]string)},
		watchers:       0,
	}

	return &e, nil
}

// NewEngine ...
func NewEngine(cfgPath string, debugMode bool) (*Engine, error) {
	var err error
	cfg, err := InitConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	return NewEngineWithConfig(cfg, debugMode)
}

// Run run run
func (e *Engine) Run() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		configName, err := writeDefaultConfig()
		if err != nil {
			log.Fatalf("Failed writing default config: %+v", err)
		}
		fmt.Printf("%s file created to the current directory with the default settings\n", configName)
		return
	}

	e.mainDebug("CWD: %s", e.config.Root)

	var err error
	if err = e.checkRunEnv(); err != nil {
		os.Exit(1)
	}
	if err = e.watching(e.config.Root); err != nil {
		os.Exit(1)
	}

	e.start()
	e.cleanup()
}

func (e *Engine) checkRunEnv() error {
	p := e.config.tmpPath()
	if _, err := os.Stat(p); os.IsNotExist(err) {
		e.runnerLog("mkdir %s", p)
		if err := os.Mkdir(p, 0o755); err != nil {
			e.runnerLog("failed to mkdir, error: %s", err.Error())
			return err
		}
	}
	return nil
}

func (e *Engine) watching(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, _ error) error {
		// NOTE: path is absolute
		if info != nil && !info.IsDir() {
			if e.checkIncludeFile(path) {
				return e.watchPath(path)
			}
			return nil
		}
		// exclude tmp dir
		if e.isTmpDir(path) {
			e.watcherLog("!exclude %s", e.config.rel(path))
			return filepath.SkipDir
		}
		// exclude testdata dir
		if e.isTestDataDir(path) {
			e.watcherLog("!exclude %s", e.config.rel(path))
			return filepath.SkipDir
		}
		// exclude hidden directories like .git, .idea, etc.
		if isHiddenDirectory(path) {
			return filepath.SkipDir
		}
		// exclude user specified directories
		if e.isExcludeDir(path) {
			e.watcherLog("!exclude %s", e.config.rel(path))
			return filepath.SkipDir
		}
		isIn, walkDir := e.checkIncludeDir(path)
		if !walkDir {
			e.watcherLog("!exclude %s", e.config.rel(path))
			return filepath.SkipDir
		}
		if isIn {
			return e.watchPath(path)
		}
		return nil
	})
}

// cacheFileChecksums calculates and stores checksums for each non-excluded file it finds from root.
func (e *Engine) cacheFileChecksums(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if info == nil {
				return err
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
			return err
		}

		if !info.Mode().IsRegular() {
			if e.isTmpDir(path) || e.isTestDataDir(path) || isHiddenDirectory(path) || e.isExcludeDir(path) {
				e.watcherDebug("!exclude checksum %s", e.config.rel(path))
				return filepath.SkipDir
			}

			// Follow symbolic link
			if e.config.Build.FollowSymlink && (info.Mode()&os.ModeSymlink) > 0 {
				link, err := filepath.EvalSymlinks(path)
				if err != nil {
					return err
				}
				linkInfo, err := os.Stat(link)
				if err != nil {
					return err
				}
				if linkInfo.IsDir() {
					err = e.watchPath(link)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}

		if e.isExcludeFile(path) || !e.isIncludeExt(path) && !e.checkIncludeFile(path) {
			e.watcherDebug("!exclude checksum %s", e.config.rel(path))
			return nil
		}

		excludeRegex, err := e.isExcludeRegex(path)
		if err != nil {
			return err
		}
		if excludeRegex {
			e.watcherDebug("!exclude checksum %s", e.config.rel(path))
			return nil
		}

		// update the checksum cache for the current file
		_ = e.isModified(path)

		return nil
	})
}

func (e *Engine) watchPath(path string) error {
	if err := e.watcher.Add(path); err != nil {
		e.watcherLog("failed to watch %s, error: %s", path, err.Error())
		return err
	}
	e.watcherLog("watching %s", e.config.rel(path))

	go func() {
		e.withLock(func() {
			e.watchers++
		})
		defer func() {
			e.withLock(func() {
				e.watchers--
			})
		}()

		if e.config.Build.ExcludeUnchanged {
			err := e.cacheFileChecksums(path)
			if err != nil {
				e.watcherLog("error building checksum cache: %v", err)
			}
		}

		for {
			select {
			case <-e.watcherStopCh:
				return
			case ev := <-e.watcher.Events():
				e.mainDebug("event: %+v", ev)
				if !validEvent(ev) {
					break
				}
				if isDir(ev.Name) {
					e.watchNewDir(ev.Name, removeEvent(ev))
					break
				}
				if e.isExcludeFile(ev.Name) {
					break
				}
				excludeRegex, _ := e.isExcludeRegex(ev.Name)
				if excludeRegex {
					break
				}
				if !e.isIncludeExt(ev.Name) && !e.checkIncludeFile(ev.Name) {
					break
				}
				e.watcherDebug("%s has changed", e.config.rel(ev.Name))
				e.eventCh <- ev.Name
			case err := <-e.watcher.Errors():
				e.watcherLog("error: %s", err.Error())
			}
		}
	}()
	return nil
}

func (e *Engine) watchNewDir(dir string, removeDir bool) {
	if e.isTmpDir(dir) {
		return
	}
	if e.isTestDataDir(dir) {
		return
	}
	if isHiddenDirectory(dir) || e.isExcludeDir(dir) {
		e.watcherLog("!exclude %s", e.config.rel(dir))
		return
	}
	if removeDir {
		if err := e.watcher.Remove(dir); err != nil {
			e.watcherLog("failed to stop watching %s, error: %s", dir, err.Error())
		}
		return
	}
	go func(dir string) {
		if err := e.watching(dir); err != nil {
			e.watcherLog("failed to watching %s, error: %s", dir, err.Error())
		}
	}(dir)
}

func (e *Engine) isModified(filename string) bool {
	newChecksum, err := fileChecksum(filename)
	if err != nil {
		e.watcherDebug("can't determine if file was changed: %v - assuming it did without updating cache", err)
		return true
	}

	if e.fileChecksums.updateFileChecksum(filename, newChecksum) {
		e.watcherDebug("stored checksum for %s: %s", e.config.rel(filename), newChecksum)
		return true
	}

	return false
}

// Endless loop and never return
func (e *Engine) start() {
	if e.config.Proxy.Enabled {
		go e.proxy.Run()
		e.mainLog("Proxy server listening on http://localhost%s", e.proxy.server.Addr)
	}

	e.running.Store(true)
	firstRunCh := make(chan bool, 1)
	firstRunCh <- true

	for {
		var filename string

		select {
		case <-e.exitCh:
			e.mainDebug("exit in start")
			return
		case filename = <-e.eventCh:
			if !e.isIncludeExt(filename) && !e.checkIncludeFile(filename) {
				continue
			}
			if e.config.Build.ExcludeUnchanged {
				if !e.isModified(filename) {
					e.mainLog("skipping %s because contents unchanged", e.config.rel(filename))
					continue
				}
			}

			// cannot set buildDelay to 0, because when the write multiple events received in short time
			// it will start Multiple buildRuns: https://github.com/air-verse/air/issues/473
			time.Sleep(e.config.buildDelay())
			e.flushEvents()

			if e.config.Screen.ClearOnRebuild {
				if e.config.Screen.KeepScroll {
					// https://stackoverflow.com/questions/22891644/how-can-i-clear-the-terminal-screen-in-go
					fmt.Print("\033[2J")
				} else {
					// https://stackoverflow.com/questions/5367068/clear-a-terminal-screen-for-real/5367075#5367075
					fmt.Print("\033c")
				}
			}

			e.mainLog("%s has changed", e.config.rel(filename))
		case <-firstRunCh:
			// go down
		}

		// already build and run now
		select {
		case <-e.buildRunCh:
			e.buildRunStopCh <- true
		default:
		}

		// if current app is running, stop it
		e.stopBin()

		go e.buildRun()
	}
}

func (e *Engine) buildRun() {
	e.buildRunCh <- true
	defer func() {
		<-e.buildRunCh
	}()

	select {
	case <-e.buildRunStopCh:
		return
	default:
	}
	var err error
	if err = e.runPreCmd(); err != nil {
		e.runnerLog("failed to execute pre_cmd: %s", err.Error())
		if e.config.Build.StopOnError {
			return
		}
	}
	if output, err := e.building(); err != nil {
		e.buildLog("failed to build, error: %s", err.Error())
		_ = e.writeBuildErrorLog(err.Error())
		if e.config.Build.StopOnError {
			// It only makes sense to run it if we stop on error. Otherwise when
			// running the binary again the error modal will be overwritten by
			// the reload.
			if e.config.Proxy.Enabled {
				e.proxy.BuildFailed(BuildFailedMsg{
					Error:   err.Error(),
					Command: e.config.Build.Cmd,
					Output:  output,
				})
			}
			return
		}
	}

	select {
	case <-e.buildRunStopCh:
		return
	case <-e.exitCh:
		e.mainDebug("exit in buildRun")
		return
	default:
	}
	if err = e.runBin(); err != nil {
		e.runnerLog("failed to run, error: %s", err.Error())
	}
}

func (e *Engine) flushEvents() {
	for {
		select {
		case <-e.eventCh:
			e.mainDebug("flushing events")
		default:
			return
		}
	}
}

// utility to execute commands, such as cmd & pre_cmd
func (e *Engine) runCommand(command string) error {
	cmd, stdout, stderr, err := e.startCmd(command)
	if err != nil {
		return err
	}
	defer func() {
		stdout.Close()
		stderr.Close()
	}()
	_, _ = io.Copy(os.Stdout, stdout)
	_, _ = io.Copy(os.Stderr, stderr)
	// wait for command to finish
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (e *Engine) runCommandCopyOutput(command string) (string, error) {
	// both stdout and stderr are piped to the same buffer, so ignore the second
	// one
	cmd, stdout, _, err := e.startCmd(command)
	if err != nil {
		return "", err
	}
	defer func() {
		stdout.Close()
	}()

	stdoutBytes, _ := io.ReadAll(stdout)
	_, _ = io.Copy(os.Stdout, strings.NewReader(string(stdoutBytes)))

	// wait for command to finish
	err = cmd.Wait()
	if err != nil {
		return string(stdoutBytes), err
	}
	return string(stdoutBytes), nil
}

// run cmd option in .air.toml
func (e *Engine) building() (string, error) {
	e.buildLog("building...")
	output, err := e.runCommandCopyOutput(e.config.Build.Cmd)
	if err != nil {
		return output, err
	}
	return output, nil
}

// run pre_cmd option in .air.toml
func (e *Engine) runPreCmd() error {
	for _, command := range e.config.Build.PreCmd {
		e.runnerLog("> %s", command)
		err := e.runCommand(command)
		if err != nil {
			return err
		}
	}
	return nil
}

// run post_cmd option in .air.toml
func (e *Engine) runPostCmd() error {
	for _, command := range e.config.Build.PostCmd {
		e.runnerLog("> %s", command)
		err := e.runCommand(command)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Engine) runBin() error {
	// killFunc returns a chan of chan of int that should be used to shutdown the bin currently being run
	// The chan int that is passed in will be used to signal completion of the shutdown
	killFunc := func(cmd *exec.Cmd, stdout io.ReadCloser, stderr io.ReadCloser, killCh chan<- struct{}, processExit <-chan struct{}) chan<- chan int {
		shutdown := make(chan chan int)
		var closer chan int

		go func() {
			defer func() {
				stdout.Close()
				stderr.Close()
			}()

			select {
			case closer = <-shutdown:
				// stopBin has been called from start or cleanup
				// defer the signalling of shutdown completion before attempting to kill further down
				defer close(closer)
				defer close(killCh)
			case <-processExit:
				// the process is exited, return
				e.withLock(func() {
					// Avoid deadlocking any racing shutdown request
					select {
					case c := <-shutdown:
						close(c)
					default:
					}
					e.binStopCh = nil
				})
				return
			}

			e.mainDebug("trying to kill pid %d, cmd %+v", cmd.Process.Pid, cmd.Args)

			pid, err := e.killCmd(cmd)
			if err != nil {
				e.mainDebug("failed to kill PID %d, error: %s", pid, err.Error())
				if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
					// Pass a non zero exit code to the closer to delegate the
					// decision wether to os.Exit or not
					closer <- 1
				}
			} else {
				e.mainDebug("cmd killed, pid: %d", pid)
			}

			if e.config.Build.StopOnError {
				cmdBinPath := cmdPath(e.config.rel(e.config.binPath()))
				if _, err = os.Stat(cmdBinPath); os.IsNotExist(err) {
					return
				}
				if err = os.Remove(cmdBinPath); err != nil {
					e.mainLog("failed to remove %s, error: %s", e.config.rel(e.config.binPath()), err)
				}
			}
		}()

		return shutdown
	}

	e.runnerLog("running...")
	go func() {

		defer func() {
			select {
			case <-e.exitCh:
				e.mainDebug("exit in runBin")
			default:
			}
		}()

		// control killFunc should be kill or not
		killCh := make(chan struct{})
		for {
			select {
			case <-killCh:
				return
			default:
				command := strings.Join(append([]string{e.config.Build.Bin}, e.runArgs...), " ")
				cmd, stdout, stderr, err := e.startCmd(command)
				if err != nil {
					e.mainLog("failed to start %s, error: %s", e.config.rel(e.config.binPath()), err.Error())
					close(killCh)
					continue
				}

				processExit := make(chan struct{})
				e.mainDebug("running process pid %v", cmd.Process.Pid)
				if e.config.Proxy.Enabled {
					e.proxy.Reload()
				}

				e.stopBin()
				e.withLock(func() {
					e.binStopCh = killFunc(cmd, stdout, stderr, killCh, processExit)
				})

				go func() {
					_, _ = io.Copy(os.Stdout, stdout)
					_, _ = cmd.Process.Wait()
				}()

				go func() {
					_, _ = io.Copy(os.Stderr, stderr)
					_, _ = cmd.Process.Wait()
				}()
				state, _ := cmd.Process.Wait()
				close(processExit)

				switch state.ExitCode() {
				case 0:
					e.runnerLog("Process Exit with Code 0")
				case -1:
					// because when we use ctrl + c to stop will return -1
				default:
					e.runnerLog("Process Exit with Code: %v", state.ExitCode())
				}

				if !e.config.Build.Rerun {
					return
				}
				time.Sleep(e.config.rerunDelay())
			}
		}
	}()

	return nil
}

func (e *Engine) stopBin() {
	e.mainDebug("initiating shutdown sequence")
	start := time.Now()
	e.mainDebug("shutdown completed in %v", time.Since(start))

	exitCode := make(chan int)

	e.withLock(func() {
		if e.binStopCh != nil {
			e.mainDebug("sending shutdown command to killfunc")
			e.binStopCh <- exitCode
			e.binStopCh = nil
		} else {
			close(exitCode)
		}
	})

	select {
	case ret := <-exitCode:
		if ret != 0 {
			e.exiter.Exit(ret) // Use exiter instead of direct os.Exit, it's for tests purpose.
		}
	case <-time.After(5 * time.Second):
		e.mainDebug("timed out waiting for process exit")
	}
}

func (e *Engine) cleanup() {
	e.mainLog("cleaning...")
	defer e.mainLog("see you again~")
	defer e.mainDebug("exited")

	if e.config.Proxy.Enabled {
		e.mainDebug("powering down the proxy...")
		if err := e.proxy.Stop(); err != nil {
			e.mainLog("failed to stop proxy: %+v", err)
		}
	}

	e.stopBin()
	e.mainDebug("waiting for close watchers..")

	e.withLock(func() {
		for i := 0; i < int(e.watchers); i++ {
			e.watcherStopCh <- true
		}
	})

	e.mainDebug("waiting for buildRun...")
	var err error
	if err = e.watcher.Close(); err != nil {
		e.mainLog("failed to close watcher, error: %s", err.Error())
	}

	e.mainDebug("waiting for clean ...")

	if e.config.Misc.CleanOnExit {
		e.mainLog("deleting %s", e.config.tmpPath())
		if err = os.RemoveAll(e.config.tmpPath()); err != nil {
			e.mainLog("failed to delete tmp dir, err: %+v", err)
		}
	}

	e.running.Store(false)
}

// Stop the air
func (e *Engine) Stop() {
	if err := e.runPostCmd(); err != nil {
		e.runnerLog("failed to execute post_cmd, error: %s", err.Error())
	}
	close(e.exitCh)
}
