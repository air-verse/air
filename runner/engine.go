package runner

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Engine ...
type Engine struct {
	config  *config
	logger  *logger
	watcher *fsnotify.Watcher

	eventCh        chan string
	watcherStopCh  chan bool
	buildRunCh     chan bool
	buildRunStopCh chan bool
	binStopCh      chan bool
	exitCh         chan bool

	mu         sync.RWMutex
	binRunning bool
	watchers   uint
}

// NewEngine ...
func NewEngine(cfgPath string) (*Engine, error) {
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

	return &Engine{
		config:         cfg,
		logger:         logger,
		watcher:        watcher,
		eventCh:        make(chan string, 1000),
		watcherStopCh:  make(chan bool, 10),
		buildRunCh:     make(chan bool, 1),
		buildRunStopCh: make(chan bool, 1),
		binStopCh:      make(chan bool),
		exitCh:         make(chan bool),
		binRunning:     false,
		watchers:       0,
	}, nil
}

// Run run run
func (e *Engine) Run() {
	var err error

	if err = e.checkRunEnv(); err != nil {
		os.Exit(1)
	}

	if err = e.watching(); err != nil {
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

func (e *Engine) watching() error {
	return filepath.Walk(e.config.watchDirRoot(), func(path string, info os.FileInfo, err error) error {
		// NOTE: path is absolute
		if !info.IsDir() {
			return nil
		}
		// exclude tmp dir
		if e.isTmpDir(path) {
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
		return e.watchDir(path)
	})
}

func (e *Engine) watchDir(path string) error {
	if err := e.watcher.Add(path); err != nil {
		e.watcherLog("failed to watching %s, error: %s", err.Error())
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

		validEvent := func(ev fsnotify.Event) bool {
			return ev.Op&fsnotify.Create == fsnotify.Create || ev.Op&fsnotify.Write == fsnotify.Write
		}
		for {
			select {
			case <-e.watcherStopCh:
				return
			case ev := <-e.watcher.Events:
				if !validEvent(ev) {
					break
				}
				if !e.isIncludeExt(ev.Name) {
					break
				}
				e.watcherLog("%s has changed", e.config.rel(ev.Name))
				e.eventCh <- ev.Name
			case err := <-e.watcher.Errors:
				e.watcherLog("error: %s", err.Error())
			}
		}
	}()
	return nil
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

	select {
	case <-e.buildRunStopCh:
		return
	default:
	}
	var err error
	err = e.building()
	if err != nil {
		e.buildLog("failed to build, error: %s", err.Error())
		e.writeBuildErrorLog(err.Error())
	}

	select {
	case <-e.buildRunStopCh:
		return
	default:
	}
	err = e.runBin()
	if err != nil {
		e.runnerLog("failed to run, error: %s", err.Error())
	}

	<-e.buildRunCh
}

func (e *Engine) flushEvents() {
	for {
		select {
		case <-e.eventCh:
		default:
			return
		}
	}
}

func (e *Engine) building() error {
	var err error

	e.buildLog("building...")
	cmd := exec.Command("/bin/sh", "-c", e.config.Build.Cmd)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}

	io.Copy(os.Stdout, stdout)
	errMsg, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}
	err = cmd.Wait()
	if err != nil {
		e := fmt.Sprintf("stderr: %s, cmd err: %s", string(errMsg), err)
		return errors.New(e)
	}
	return nil
}

func (e *Engine) runBin() error {
	var err error

	e.runnerLog("running...")
	cmd := exec.Command("/bin/sh", "-c", e.config.binPath())
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	err = cmd.Start()
	if err != nil {
		return err
	}

	e.withLock(func() {
		e.binRunning = true
	})

	aw := newAppLogWriter(e.logger)
	go io.Copy(aw, stderr)
	go io.Copy(aw, stdout)

	go func() {
		<-e.binStopCh
		pid, err := killCmd(cmd)
		if err != nil {
			e.mainLog("failed to kill PID %d, error: %s", pid, err.Error())
			os.Exit(1)
		}
		e.withLock(func() {
			e.binRunning = false
		})
	}()
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
}

// Stop the air
func (e *Engine) Stop() {
	e.exitCh <- true
}
