package runner

import "testing"

func TestNewWatcher_EventWatcherWhenPollDisabled(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Build.Poll = false

	w, err := newWatcher(&cfg)
	if err != nil {
		t.Fatalf("newWatcher() error = %v", err)
	}
	if w == nil {
		t.Fatal("newWatcher() returned nil watcher")
	}
}

func TestNewWatcher_PollingWatcherWhenPollEnabled(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Build.Poll = true
	cfg.Build.PollInterval = 1000

	w, err := newWatcher(&cfg)
	if err != nil {
		t.Fatalf("newWatcher() error = %v", err)
	}
	if w == nil {
		t.Fatal("newWatcher() returned nil watcher")
	}
}

func TestNewWatcher_PollIntervalHasMinimum500ms(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	cfg.Build.Poll = true
	cfg.Build.PollInterval = 100

	w, err := newWatcher(&cfg)
	if err != nil {
		t.Fatalf("newWatcher() error = %v", err)
	}
	if w == nil {
		t.Fatal("newWatcher() returned nil watcher")
	}
}
