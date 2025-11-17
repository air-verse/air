package runner

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

const (
	sliceCmdArgSeparator = ","
)

func (e *Engine) mainLog(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	e.logWithLock(func() {
		e.logger.main()(format, v...)
	})
}

func (e *Engine) mainDebug(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	if e.debugMode {
		e.mainLog(format, v...)
	}
}

func (e *Engine) buildLog(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	if e.debugMode || !e.config.Log.MainOnly {
		e.logWithLock(func() {
			e.logger.build()(format, v...)
		})
	}
}

func (e *Engine) runnerLog(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	if e.debugMode || !e.config.Log.MainOnly {
		e.logWithLock(func() {
			e.logger.runner()(format, v...)
		})
	}
}

func (e *Engine) watcherLog(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	if e.debugMode || !e.config.Log.MainOnly {
		e.logWithLock(func() {
			e.logger.watcher()(format, v...)
		})
	}
}

func (e *Engine) watcherDebug(format string, v ...interface{}) {
	if e.config.Log.Silent {
		return
	}
	if e.debugMode {
		e.watcherLog(format, v...)
	}
}

func (e *Engine) isTmpDir(path string) bool {
	return path == e.config.tmpPath()
}

func (e *Engine) isTestDataDir(path string) bool {
	return path == e.config.testDataPath()
}

func isHiddenDirectory(path string) bool {
	return len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".") && filepath.Base(path) != ".."
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

func (e *Engine) checkIncludeFile(path string) bool {
	cleanName := cleanPath(e.config.rel(path))
	iFile := e.config.Build.IncludeFile
	if len(iFile) == 0 { // ignore empty
		return false
	}
	if cleanName == "." {
		return false
	}
	for _, d := range iFile {
		if d == cleanName {
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

func (e *Engine) isExcludeRegex(path string) (bool, error) {
	regexes, err := e.config.Build.RegexCompiled()
	if err != nil {
		return false, err
	}
	for _, re := range regexes {
		if re.Match([]byte(path)) {
			return true, nil
		}
	}
	return false, nil
}

func (e *Engine) isExcludeFile(path string) bool {
	cleanName := cleanPath(e.config.rel(path))
	for _, d := range e.config.Build.ExcludeFile {
		matched, err := filepath.Match(d, cleanName)
		if err == nil && matched {
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

func copyOutput(dst io.Writer, src io.Reader) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		_, _ = dst.Write([]byte(scanner.Text() + "\n"))
	}
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

// splitBinArgs splits a bin string into the binary path and its arguments.
// This handles the legacy case where Build.Bin may contain both path and args as a space-separated string.
//
// For Windows absolute paths (e.g., C:\Program Files\app.exe), it detects the executable
// by looking for common Windows executable extensions (.exe, .bat, .cmd, .com).
// For Unix-like absolute paths (starting with /), it uses smart detection:
//   - Looks for flags (--flag or -f) which clearly mark start of arguments
//   - If no flags, splits at the first space-separated part that doesn't contain '/'
//   - This handles paths like "/path with spaces/app serve :9898" correctly
//
// For relative paths and simple names, splits on first space unless remaining contains '/'.
//
// Returns (binaryPath, []arguments)
func splitBinArgs(bin string) (string, []string) {
	// Check if this looks like a Windows absolute path (e.g., C:\... or C:/...)
	// Windows absolute paths start with a drive letter followed by colon
	windowsPathPattern := regexp.MustCompile(`^[a-zA-Z]:[/\\]`)

	if windowsPathPattern.MatchString(bin) {
		// For Windows paths, try to find the executable extension
		// Common extensions: .exe, .bat, .cmd, .com
		exePattern := regexp.MustCompile(`(?i)\.(exe|bat|cmd|com)(\s|$)`)
		matches := exePattern.FindStringIndex(bin)

		if matches != nil {
			// Found an extension, split after it
			splitPoint := matches[1]
			binPath := strings.TrimSpace(bin[:splitPoint])
			remaining := strings.TrimSpace(bin[splitPoint:])
			if remaining == "" {
				return binPath, nil
			}
			args := strings.Fields(remaining)
			return binPath, args
		}
		// No extension found, check if there's a space
		// If no space, return the whole thing as the path
		if !strings.Contains(bin, " ") {
			return bin, nil
		}
		// If there's a space but no extension, we can't reliably split
		// Fall through to Unix-style detection
	}

	// For Unix paths
	if !strings.Contains(bin, " ") {
		return bin, nil
	}

	// For Unix absolute paths, use smarter detection
	if strings.HasPrefix(bin, "/") {
		// It's an absolute Unix path
		// Look for flags first - they clearly mark the start of arguments
		flagPattern := regexp.MustCompile(`\s+--?[a-zA-Z]`)
		flagMatch := flagPattern.FindStringIndex(bin)

		if flagMatch != nil {
			// Split at the flag position
			binPath := strings.TrimSpace(bin[:flagMatch[0]])
			remainingArgs := strings.TrimSpace(bin[flagMatch[0]:])
			args := strings.Fields(remainingArgs)
			return binPath, args
		}

		// No flags found. Try to find where the path ends.
		// Split by spaces and find the first part that doesn't contain /
		// Everything before that part is the path, from there on are args
		parts := strings.Fields(bin)

		for i := 1; i < len(parts); i++ {
			if !strings.Contains(parts[i], "/") {
				// This part doesn't contain /, likely start of args
				// Rejoin: parts[0:i] as path, parts[i:] as args
				pathParts := parts[0:i]
				path := strings.Join(pathParts, " ")
				args := parts[i:]
				return path, args
			}
		}

		// All parts contain / or it's a single path, so it's all path
		return bin, nil
	}

	// Not an absolute path, use simple space-based split
	parts := strings.SplitN(bin, " ", 2)
	if len(parts) == 1 {
		return parts[0], nil
	}

	remaining := strings.TrimSpace(parts[1])

	// Check if remaining part looks like path components (contains /)
	if strings.Contains(remaining, "/") {
		// Likely more path
		return bin, nil
	}

	// At this point: remaining doesn't contain '/'
	// This is the legacy format: "binary cmdname" or "binary arg1 arg2"
	args := strings.Fields(remaining)
	return parts[0], args
}

func adaptToVariousPlatforms(c *Config) {
	// Fix the default configuration is not used in Windows
	// Use the unix configuration on Windows
	if runtime.GOOS == PlatformWindows {

		runName := "start"
		extName := ".exe"
		originBin := c.Build.Bin

		if 0 < len(c.Build.FullBin) {

			if !strings.HasSuffix(c.Build.FullBin, extName) {
				c.Build.FullBin += extName
			}
			if !strings.HasPrefix(c.Build.FullBin, runName) {
				c.Build.FullBin = runName + " /wait /b " + c.Build.FullBin
			}
		}

		// bin=/tmp/main  cmd=go build -o ./tmp/main.exe main.go
		if !strings.Contains(c.Build.Cmd, c.Build.Bin) && strings.Contains(c.Build.Cmd, originBin) {
			c.Build.Cmd = strings.Replace(c.Build.Cmd, originBin, c.Build.Bin, 1)
		}
	}
}

// fileChecksum returns a checksum for the given file's contents.
func fileChecksum(filename string) (checksum string, err error) {
	contents, err := os.ReadFile(filename)
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

// updateFileChecksum updates the filename with the given checksum if different.
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

// TomlInfo is a struct for toml config file
type TomlInfo struct {
	fieldPath  string
	field      reflect.StructField
	Value      *string
	fieldValue string
	usage      string
}

func setValue2Struct(v reflect.Value, fieldName string, value string) {
	index := strings.Index(fieldName, ".")
	if index == -1 && len(fieldName) == 0 {
		return
	}
	fields := strings.Split(fieldName, ".")
	var addressableVal reflect.Value
	switch v.Type().String() {
	case "*runner.Config":
		addressableVal = v.Elem()
	default:
		addressableVal = v
	}
	if len(fields) == 1 {
		// string slice int switch case
		field := addressableVal.FieldByName(fieldName)
		switch field.Kind() {
		case reflect.String:
			field.SetString(value)
		case reflect.Slice:
			if len(value) == 0 {
				field.Set(reflect.ValueOf([]string{}))
			} else {
				field.Set(reflect.ValueOf(strings.Split(value, sliceCmdArgSeparator)))
			}
		case reflect.Int64:
			i, _ := strconv.ParseInt(value, 10, 64)
			field.SetInt(i)
		case reflect.Int:
			i, _ := strconv.Atoi(value)
			field.SetInt(int64(i))
		case reflect.Bool:
			b, _ := strconv.ParseBool(value)
			field.SetBool(b)
		case reflect.Ptr:
			field.SetString(value)
		default:
			log.Fatalf("unsupported type %s", v.FieldByName(fields[0]).Kind())
		}
	} else if len(fields) == 0 {
		return
	} else {
		field := addressableVal.FieldByName(fields[0])
		s2 := fieldName[index+1:]
		setValue2Struct(field, s2, value)
	}
}

// flatConfig ...
func flatConfig(stut interface{}) map[string]TomlInfo {
	m := make(map[string]TomlInfo)
	t := reflect.TypeOf(stut)
	v := reflect.ValueOf(stut)
	setTage2Map("", t, v, m, "")
	return m
}

func getFieldValueString(fieldValue reflect.Value) string {
	switch fieldValue.Kind() {
	case reflect.Slice:
		sliceLen := fieldValue.Len()
		strSlice := make([]string, sliceLen)
		for j := 0; j < sliceLen; j++ {
			strSlice[j] = fmt.Sprintf("%v", fieldValue.Index(j).Interface())
		}
		return strings.Join(strSlice, ",")
	default:
		return fmt.Sprintf("%v", fieldValue.Interface())
	}
}

func setTage2Map(root string, t reflect.Type, v reflect.Value, m map[string]TomlInfo, fieldPath string) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		tomlVal := field.Tag.Get("toml")

		if field.Type.Kind() == reflect.Struct {
			path := fieldPath + field.Name + "."
			setTage2Map(root+tomlVal+".", field.Type, fieldValue, m, path)
			continue
		}

		if tomlVal == "" {
			continue
		}

		tomlPath := root + tomlVal
		path := fieldPath + field.Name
		var v *string
		str := ""
		v = &str

		fieldValueStr := getFieldValueString(fieldValue)
		usage := field.Tag.Get("usage")
		m[tomlPath] = TomlInfo{field: field, Value: v, fieldPath: path, fieldValue: fieldValueStr, usage: usage}
	}
}

func joinPath(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(root, path)
}

func formatPath(path string) string {
	if !filepath.IsAbs(path) || !strings.Contains(path, " ") {
		return path
	}

	quotedPath := fmt.Sprintf(`"%s"`, path)

	if runtime.GOOS == PlatformWindows {
		return fmt.Sprintf(`& %s`, quotedPath)
	}

	return quotedPath
}
