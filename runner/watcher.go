package runner

import (
	"time"

	"github.com/gohugoio/hugo/watcher/filenotify"
)

func newWatcher(cfg *Config) (filenotify.FileWatcher, error) {
	if !cfg.Build.Poll {
		return filenotify.NewEventWatcher()
	}

	// Get the poll interval from the config.
	interval := cfg.Build.Delay

	// Configure a minimum poll interval of 500ms.
	if interval < 500 {
		interval = 500
	}
	pollInterval := time.Duration(interval) * time.Millisecond

	return filenotify.NewPollingWatcher(pollInterval), nil
}
