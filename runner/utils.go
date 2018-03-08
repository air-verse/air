package runner

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/fsnotify/fsnotify"
)

func (e *Engine) mainLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.main()(format, v...)
	})
}

func (e *Engine) mainDebug(format string, v ...interface{}) {
	if e.debugMode {
		e.mainLog(format, v...)
	}
}

func (e *Engine) buildLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.build()(format, v...)
	})
}

func (e *Engine) buildDebug(format string, v ...interface{}) {
	if e.debugMode {
		e.buildLog(format, v...)
	}
}

func (e *Engine) runnerLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.runner()(format, v...)
	})
}

func (e *Engine) runnerDebug(format string, v ...interface{}) {
	if e.debugMode {
		e.runnerLog(format, v...)
	}
}

func (e *Engine) watcherLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.watcher()(format, v...)
	})
}

func (e *Engine) watcherDebug(format string, v ...interface{}) {
	if e.debugMode {
		e.watcherLog(format, v...)
	}
}

func (e *Engine) appLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.app()(format, v...)
	})
}

func (e *Engine) appDebug(format string, v ...interface{}) {
	if e.debugMode {
		e.appLog(format, v...)
	}
}

func (e *Engine) isTmpDir(path string) bool {
	return path == e.config.tmpPath()
}

func isHiddenDirectory(path string) bool {
	return len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".")
}

func cleanPath(path string) string {
	return strings.TrimSuffix(strings.TrimSpace(path), "/")
}

func (e *Engine) isExcludeDir(path string) bool {
	rp := e.config.rel(path)
	for _, d := range e.config.Build.ExcludeDir {
		if cleanPath(rp) == d {
			return true
		}
	}
	return false
}

func (e *Engine) isIncludeExt(path string) bool {
	ext := filepath.Ext(path)
	for _, v := range e.config.Build.IncludeExt {
		if ext == "."+strings.TrimSpace(v) {
			return true
		}
	}
	return false
}

func (e *Engine) writeBuildErrorLog(msg string) error {
	var err error
	f, err := os.OpenFile(e.config.buildLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	if _, err = f.Write([]byte(msg)); err != nil {
		return err
	}
	return f.Close()
}

func (e *Engine) withLock(f func()) {
	e.mu.Lock()
	f()
	e.mu.Unlock()
}

func (e *Engine) logWithLock(f func()) {
	e.ll.Lock()
	f()
	e.ll.Unlock()
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home := os.Getenv("HOME")
		return home + path[1:], nil
	}
	var err error
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if path == "." {
		return wd, nil
	}
	if strings.HasPrefix(path, "./") {
		return wd + path[1:], nil
	}
	return path, nil
}

func killCmd(cmd *exec.Cmd) (int, error) {
	pid := cmd.Process.Pid
	// https://stackoverflow.com/a/44551450
	if runtime.GOOS == "windows" {
		kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(pid))
		return pid, kill.Run()
	}
	// https://stackoverflow.com/questions/22470193/why-wont-go-kill-a-child-process-correctly
	if runtime.GOOS == "linux" {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		return pid, err
	}
	return pid, cmd.Process.Kill()
}

func isDir(path string) bool {
	i, err := os.Stat(path)
	if err != nil {
		return false
	}
	return i.IsDir()
}

func validEvent(ev fsnotify.Event) bool {
	return ev.Op&fsnotify.Create == fsnotify.Create ||
		ev.Op&fsnotify.Write == fsnotify.Write ||
		ev.Op&fsnotify.Remove == fsnotify.Remove
}

func removeEvent(ev fsnotify.Event) bool {
	return ev.Op&fsnotify.Remove == fsnotify.Remove
}

func (e *Engine) startCmd(cmd string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	var err error
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.Command("cmd", "/c", cmd)
	} else {
		c = exec.Command("/bin/sh", "-c", cmd)
		c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}
	err = c.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	return c, stdout, stderr, err
}

func cmdPath(path string) string {
	return strings.Split(path, " ")[0]
}
