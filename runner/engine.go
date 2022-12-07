package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Engine ...
type Engine struct {
	ctx       context.Context
	config    *Config
	logger    *logger
	watcher   *fsnotify.Watcher
	debugMode bool
	runArgs   []string
	running   bool

	eventCh        chan string
	watcherStopCh  chan bool
	buildRunCh     chan bool
	buildRunStopCh chan bool
	canExit        chan bool
	binStopCh      chan bool
	exitCh         chan bool

	mu            sync.RWMutex
	watchers      uint
	fileChecksums *checksumMap

	ll sync.Mutex // lock for logger
}

// NewEngineWithConfig ...
func NewEngineWithConfig(cfg *Config, debugMode bool) (*Engine, error) {
	logger := newLogger(cfg)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	e := Engine{
		ctx:            context.Background(),
		config:         cfg,
		logger:         logger,
		watcher:        watcher,
		debugMode:      debugMode,
		runArgs:        cfg.Build.ArgsBin,
		eventCh:        make(chan string, 1000),
		watcherStopCh:  make(chan bool, 10),
		buildRunCh:     make(chan bool, 1),
		buildRunStopCh: make(chan bool, 1),
		canExit:        make(chan bool, 1),
		binStopCh:      make(chan bool),
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
		writeDefaultConfig()
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
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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

		if e.isExcludeFile(path) || !e.isIncludeExt(path) {
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
				excludeRegex, _ := e.isExcludeRegex(ev.Name)
				if excludeRegex {
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
	e.running = true
	firstRunCh := make(chan bool, 1)
	firstRunCh <- true
	var buildRunCancel context.CancelFunc

	for {
		var filename string

		select {
		case <-e.exitCh:
			e.mainDebug("exit in start")
			return
		case filename = <-e.eventCh:
			if !e.isIncludeExt(filename) {
				continue
			}
			if e.config.Build.ExcludeUnchanged {
				if !e.isModified(filename) {
					e.mainLog("skipping %s because contents unchanged", e.config.rel(filename))
					continue
				}
			}

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
			break
		}

		// cancel the original buildRun goRoutine
		if buildRunCancel != nil {
			buildRunCancel()
		}

		buildRunCtx, cancel := context.WithCancel(e.ctx)
		buildRunCancel = cancel
		go e.buildRun(buildRunCtx)
	}
}

func (e *Engine) doBuild(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		if err := e.building(ctx); err != nil {
			_ = e.writeBuildErrorLog(err.Error())
			return
		}
	}
}

func (e *Engine) doRunBin(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		if err := e.runBin(ctx); err != nil {
			e.runnerLog("failed to run, error: %s", err.Error())
		}
	}
}

func (e *Engine) buildRun(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		e.doBuild(ctx)
		e.doRunBin(ctx)
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

func (e *Engine) building(ctx context.Context) error {
	var err error
	e.buildLog("building...")
	cmd, stdout, stderr, err := e.startCmd(ctx, e.config.Build.Cmd)
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

func (e *Engine) runBin(ctx context.Context) error {
	e.runnerLog("running...")
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			command := strings.Join(append([]string{e.config.Build.Bin}, e.runArgs...), " ")
			cmd, stdout, stderr, err := e.startCmd(ctx, command)
			if err != nil {
				return err
			}
			_, _ = io.Copy(os.Stdout, stdout)
			_, _ = io.Copy(os.Stderr, stderr)
			_, _ = cmd.Process.Wait()
			// run delay
			time.Sleep(1 * time.Second)
		}
	}
}

func (e *Engine) cleanup() {
	e.mainLog("cleaning...")
	defer e.mainLog("see you again~")

	e.mainDebug("wating for	close watchers..")
	e.withLock(func() {
		for i := 0; i < int(e.watchers); i++ {
			e.watcherStopCh <- true
		}
	})

	var err error
	if err = e.watcher.Close(); err != nil {
		e.mainLog("failed to close watcher, error: %s", err.Error())
	}

	e.mainDebug("waiting for buildRun...")
	_, cancel := context.WithCancel(e.ctx)
	cancel()

	e.mainDebug("waiting for clean ...")
	if e.config.Misc.CleanOnExit {
		e.mainLog("deleting %s", e.config.tmpPath())
		if err = os.RemoveAll(e.config.tmpPath()); err != nil {
			e.mainLog("failed to delete tmp dir, err: %+v", err)
		}
	}

	e.mainDebug("exited")
}

// Stop the air
func (e *Engine) Stop() {
	close(e.exitCh)
}
