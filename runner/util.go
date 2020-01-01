package runner

import (
	"os"
	"path/filepath"
	"strings"

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
	cleanName := cleanPath(e.config.rel(path))
	for _, d := range e.config.Build.ExcludeDir {
		if cleanName == d {
			return true
		}
	}
	return false
}

// return isIncludeDir, walkDir
func (e *Engine) checkIncludeDir(path string) (bool, bool) {
	cleanName := cleanPath(e.config.rel(path))
	iDir := e.config.Build.IncludeDir
	if len(iDir) == 0 { // ignore empty
		return true, true
	}
	if cleanName == "." {
		return false, true
	}
	walkDir := false
	for _, d := range iDir {
		if d == cleanName {
			return true, true
		}
		if strings.HasPrefix(cleanName, d) { // current dir is sub-directory of `d`
			return true, true
		}
		if strings.HasPrefix(d, cleanName) { // `d` is sub-directory of current dir
			walkDir = true
		}
	}
	return false, walkDir
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

func (e *Engine) isExcludeFile(path string) bool {
	cleanName := cleanPath(e.config.rel(path))
	for _, d := range e.config.Build.ExcludeFile {
		if d == cleanName {
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

func cmdPath(path string) string {
	return strings.Split(path, " ")[0]
}
