package runner

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// matchRuleIndex returns the index of the first rule matching path, or -1.
// Rules are checked before the main build filters, so a file matched by a
// rule never triggers a rebuild.
func (e *Engine) matchRuleIndex(path string) int {
	for i := range e.config.Build.Rules {
		if e.ruleMatches(&e.config.Build.Rules[i], path) {
			return i
		}
	}
	return -1
}

func (e *Engine) ruleMatches(r *cfgRule, path string) bool {
	if len(r.includeDirAbs) > 0 {
		inDir := false
		cleaned := filepath.Clean(path)
		for _, dir := range r.includeDirAbs {
			if isSubPath(dir, cleaned) {
				inDir = true
				break
			}
		}
		if !inDir {
			return false
		}
	}
	for _, re := range r.regexCompiled {
		if re.MatchString(path) {
			return false
		}
	}
	if len(r.IncludeExt) == 0 && len(r.IncludeFile) == 0 {
		return true
	}
	ext := filepath.Ext(path)
	for _, v := range r.IncludeExt {
		v = strings.TrimSpace(v)
		if v == extWildcard || ext == "."+v {
			return true
		}
	}
	rel := cleanPath(e.config.rel(path))
	for _, f := range r.IncludeFile {
		if f == rel {
			return true
		}
	}
	return false
}

// isRuleDir reports whether dir is exactly a rule's include_dir. Such dirs
// are watched even when the main build excludes them via exclude_dir.
func (e *Engine) isRuleDir(dir string) bool {
	cleaned := filepath.Clean(dir)
	for i := range e.config.Build.Rules {
		for _, d := range e.config.Build.Rules[i].includeDirAbs {
			if d == cleaned {
				return true
			}
		}
	}
	return false
}

// inRuleDir reports whether dir is a rule's include_dir or inside one.
func (e *Engine) inRuleDir(dir string) bool {
	cleaned := filepath.Clean(dir)
	for i := range e.config.Build.Rules {
		for _, d := range e.config.Build.Rules[i].includeDirAbs {
			if isSubPath(d, cleaned) {
				return true
			}
		}
	}
	return false
}

func (e *Engine) ruleLog(name string, format string, v ...interface{}) {
	e.runnerLog("[%s] %s", name, fmt.Sprintf(format, v...))
}

// runRule consumes change events for one rule, debounces them, and runs the
// rule's cmd. The cmd runs to completion; events arriving meanwhile stay
// queued and trigger another run afterwards.
func (e *Engine) runRule(idx int) {
	rule := &e.config.Build.Rules[idx]
	ch := e.ruleEventChs[idx]
	for {
		select {
		case <-e.exitCh:
			return
		case filename := <-ch:
			time.Sleep(rule.delay())
			// coalesce the burst of events into a single run
			for drained := false; !drained; {
				select {
				case <-ch:
				default:
					drained = true
				}
			}
			e.ruleLog(rule.Name, "%s has changed", e.config.rel(filename))
			e.ruleLog(rule.Name, "> %s", rule.Cmd)
			if err := e.runCommand(rule.Cmd); err != nil {
				e.ruleLog(rule.Name, "failed to execute cmd: %s", err.Error())
			}
		}
	}
}
