package runner

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Engine ...
type Engine struct {
	config    *config
	logger    *logger
	watcher   *fsnotify.Watcher
	debugMode bool

	eventCh        chan string
	watcherStopCh  chan bool
	buildRunCh     chan bool
	buildRunStopCh chan bool
	binStopCh      chan bool
	exitCh         chan bool

	mu            sync.RWMutex
	binRunning    bool
	watchers      uint
	fileChecksums *checksumMap

	ll sync.Mutex // lock for logger
}

// NewEngine ...
func NewEngine(cfgPath string, debugMode bool) (*Engine, error) {
	var err error
	cfg, err := initConfig(cfgPath)
	if err != nil {
		return nil, err
	}

	logger := newLogger(cfg)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	e := Engine{
		config:         cfg,
		logger:         logger,
		watcher:        watcher,
		debugMode:      debugMode,
		eventCh:        make(chan string, 1000),
		watcherStopCh:  make(chan bool, 10),
		buildRunCh:     make(chan bool, 1),
		buildRunStopCh: make(chan bool, 1),
		binStopCh:      make(chan bool),
		exitCh:         make(chan bool),
		binRunning:     false,
		watchers:       0,
	}

	if cfg.Build.ExcludeUnchanged {
		e.fileChecksums = &checksumMap{m: make(map[string]string)}
	}

	return &e, nil
}

// Run run run
func (e *Engine) Run() {
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
		if err := os.Mkdir(p, 0755); err != nil {
			e.runnerLog("failed to mkdir, error: %s", err.Error())
			return err
		}
	}
	return nil
}

func (e *Engine) watching(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// NOTE: path is absolute
		if info != nil && !info.IsDir() {
			return nil
		}
		// exclude tmp dir
		if e.isTmpDir(path) {
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
			return e.watchDir(path)
		}
		return nil
	})
}

// cacheFileChecksums calculates and stores checksums for each non-excluded file it finds from root.
func (e *Engine) cacheFileChecksums(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return err
		}

		if !info.Mode().IsRegular() {
			if e.isTmpDir(path) || isHiddenDirectory(path) || e.isExcludeDir(path) {
				e.watcherDebug("!exclude checksum %s", e.config.rel(path))
				return filepath.SkipDir
			}
		}

		if e.isExcludeFile(path) || !e.isIncludeExt(path) {
			e.watcherDebug("!exclude checksum %s", e.config.rel(path))
			return nil
		}

		// update the checksum cache for the current file
		_ = e.isModified(path)

		return nil
	})
}

func (e *Engine) watchDir(path string) error {
	if err := e.watcher.Add(path); err != nil {
		e.watcherLog("failed to watching %s, error: %s", path, err.Error())
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
			case ev := <-e.watcher.Events:
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
				if !e.isIncludeExt(ev.Name) {
					break
				}
				e.watcherDebug("%s has changed", e.config.rel(ev.Name))
				e.eventCh <- ev.Name
			case err := <-e.watcher.Errors:
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
	firstRunCh := make(chan bool, 1)
	firstRunCh <- true

	for {
		var filename string

		select {
		case <-e.exitCh:
			return
		case filename = <-e.eventCh:
			time.Sleep(e.config.buildDelay())
			e.flushEvents()
			if !e.isIncludeExt(filename) {
				continue
			}
			if e.config.Build.ExcludeUnchanged {
				if !e.isModified(filename) {
					e.mainLog("skipping %s because contents unchanged", e.config.rel(filename))
					continue
				}
			}
			e.mainLog("%s has changed", e.config.rel(filename))
		case <-firstRunCh:
			// go down
			break
		}

		select {
		case <-e.buildRunCh:
			e.buildRunStopCh <- true
		default:
		}
		e.withLock(func() {
			if e.binRunning {
				e.binStopCh <- true
			}
		})
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
	if err = e.building(); err != nil {
		e.buildLog("failed to build, error: %s", err.Error())
		_ = e.writeBuildErrorLog(err.Error())
		if e.config.Build.StopOnError {
			return
		}
	}

	select {
	case <-e.buildRunStopCh:
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

func (e *Engine) building() error {
	var err error
	e.buildLog("building...")
	cmd, stdout, stderr, err := e.startCmd(e.config.Build.Cmd)
	if err != nil {
		return err
	}
	defer func() {
		stdout.Close()
		stderr.Close()
	}()
	_, _ = io.Copy(os.Stdout, stdout)
	_, _ = io.Copy(os.Stderr, stderr)
	// wait for building
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (e *Engine) runBin() error {
	var err error
	e.runnerLog("running...")
	cmd, stdout, stderr, err := e.startCmd(e.config.Build.Bin)
	if err != nil {
		return err
	}
	e.withLock(func() {
		e.binRunning = true
	})

	go func() {
		_, _ = io.Copy(os.Stdout, stdout)
		_, _ = io.Copy(os.Stderr, stderr)
	}()

	go func(cmd *exec.Cmd, stdout io.ReadCloser, stderr io.ReadCloser) {
		<-e.binStopCh
		e.mainDebug("trying to kill cmd %+v", cmd.Args)
		defer func() {
			stdout.Close()
			stderr.Close()
		}()

		var err error
		pid, err := e.killCmd(cmd)
		if err != nil {
			e.mainDebug("failed to kill PID %d, error: %s", pid, err.Error())
			if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
				os.Exit(1)
			}
		} else {
			e.mainDebug("cmd killed, pid: %d", pid)
		}
		e.withLock(func() {
			e.binRunning = false
		})
		cmdBinPath := cmdPath(e.config.rel(e.config.binPath()))
		if _, err = os.Stat(cmdBinPath); os.IsNotExist(err) {
			return
		}
		if err = os.Remove(cmdBinPath); err != nil {
			e.mainLog("failed to remove %s, error: %s", e.config.rel(e.config.binPath()), err)
		}
	}(cmd, stdout, stderr)
	return nil
}

func (e *Engine) cleanup() {
	e.mainLog("cleaning...")
	defer e.mainLog("see you again~")

	e.withLock(func() {
		if e.binRunning {
			e.binStopCh <- true
		}
	})

	e.withLock(func() {
		for i := 0; i < int(e.watchers); i++ {
			e.watcherStopCh <- true
		}
	})

	var err error
	if err = e.watcher.Close(); err != nil {
		e.mainLog("failed to close watcher, error: %s", err.Error())
	}

	if e.config.Misc.CleanOnExit {
		e.mainLog("deleting %s", e.config.tmpPath())
		if err = os.RemoveAll(e.config.tmpPath()); err != nil {
			e.mainLog("failed to delete tmp dir, err: %+v", err)
		}
	}
}

// Stop the air
func (e *Engine) Stop() {
	e.exitCh <- true
}
