package runner

import (
	"os"
	"path/filepath"
	"strings"
)

func (e *Engine) mainLog(format string, v ...interface{}) {
	e.logger.Main()(format, v...)
}

func (e *Engine) buildLog(format string, v ...interface{}) {
	e.logger.Build()(format, v...)
}

func (e *Engine) runnerLog(format string, v ...interface{}) {
	e.logger.Runner()(format, v...)
}

func (e *Engine) watcherLog(format string, v ...interface{}) {
	e.logger.Watcher()(format, v...)
}

func (e *Engine) appLog(format string, v ...interface{}) {
	e.logger.App()(format, v...)
}

func (e *Engine) isTmpDir(path string) bool {
	absTmpPath, _ := filepath.Abs(e.config.TmpPath)
	absPath, _ := filepath.Abs(path)
	return absTmpPath == absPath
}

func isHiddenDirectory(path string) bool {
	return len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".")
}

func cleanPath(path string) string {
	return strings.TrimSuffix(strings.TrimSpace(path), "/")
}

func (e *Engine) isExcludeDir(path string) bool {
	for _, d := range e.config.Build.ExcludeDir {
		if cleanPath(path) == d {
			return true
		}
	}
	return false
}

func (e *Engine) isIncludeExt(path string) bool {
	ext := filepath.Ext(path)
	for _, v := range e.config.Build.IncludeExt {
		if ext == "." + strings.TrimSpace(v) {
			return true
		}
	}
	return false
}

func (e *Engine) writeBuildErrorLog(msg string) error {
	var err error
	f, err := os.OpenFile(e.config.Build.Log, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
