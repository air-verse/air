package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

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

func (e *Engine) runnerLog(format string, v ...interface{}) {
	e.logWithLock(func() {
		e.logger.runner()(format, v...)
	})
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

func addArgs(bin string, args []string) string {
	for _, arg := range args {
		bin += fmt.Sprintf(" %s", arg)
	}

	return bin
}

func adaptToVariousPlatforms(c *config) {
	// Fix the default configuration is not used in Windows
	// Use the unix configuration on Windows
	if runtime.GOOS == PlatformWindows {

		runName := "start"
		extName := ".exe"
		originBin := c.Build.Bin
		if !strings.HasSuffix(c.Build.Bin, extName) {

			c.Build.Bin += extName
		}

		if 0 < len(c.Build.FullBin) {

			if !strings.HasSuffix(c.Build.FullBin, extName) {

				c.Build.FullBin += extName
			}

			if !strings.HasPrefix(c.Build.FullBin, runName) {
				c.Build.FullBin = runName + " /b " + c.Build.FullBin
			}

			c.Build.FullBin = addArgs(c.Build.FullBin, c.Build.ArgsBin)
		}

		// bin=/tmp/main  cmd=go build -o ./tmp/main.exe main.go
		if !strings.Contains(c.Build.Cmd, c.Build.Bin) && strings.Contains(c.Build.Cmd, originBin) {
			c.Build.Cmd = strings.Replace(c.Build.Cmd, originBin, c.Build.Bin, 1)
		}

		c.Build.Bin = addArgs(c.Build.Bin, c.Build.ArgsBin)
	}
}

// fileChecksum returns a checksum for the given file's contents.
func fileChecksum(filename string) (checksum string, err error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}

	// If the file is empty, an editor might've been in the process of rewriting the file when we read it.
	// This can happen often if editors are configured to run format after save.
	// Instead of calculating a new checksum, we'll assume the file was unchanged, but return an error to force a rebuild anyway.
	if len(contents) == 0 {
		return "", errors.New("empty file, forcing rebuild without updating checksum")
	}

	h := sha256.New()
	if _, err := h.Write(contents); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// checksumMap is a thread-safe map to store file checksums.
type checksumMap struct {
	l sync.Mutex
	m map[string]string
}

// update updates the filename with the given checksum if different.
func (a *checksumMap) updateFileChecksum(filename, newChecksum string) (ok bool) {
	a.l.Lock()
	defer a.l.Unlock()
	oldChecksum, ok := a.m[filename]
	if !ok || oldChecksum != newChecksum {
		a.m[filename] = newChecksum
		return true
	}
	return false
}
