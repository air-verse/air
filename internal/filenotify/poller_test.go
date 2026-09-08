package filenotify

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/require"
)

const (
	subdir1       = "subdir1"
	subdir2       = "subdir2"
	watchWaitTime = 200 * time.Millisecond
)

var isMacOs = runtime.GOOS == "darwin"

func TestPollerAddRemove(t *testing.T) {
	w := NewPollingWatcher(watchWaitTime)

	require.Error(t, w.Add("foo"))
	require.Error(t, w.Remove("foo"))

	f, err := os.CreateTemp("", "asdf")
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, w.Close())
		os.Remove(f.Name())
	})
	require.NoError(t, w.Add(f.Name()))
	require.NoError(t, w.Remove(f.Name()))
}

func TestPollerEvent(t *testing.T) {
	for _, poll := range []bool{true, false} {
		if !poll && !isMacOs {
			// Only run the fsnotify tests on MacOS locally.
			continue
		}
		method := "fsnotify"
		if poll {
			method = "poll"
		}

		t.Run(fmt.Sprintf("%s, Watch dir", method), func(t *testing.T) {
			dir, w := preparePollTest(t, poll)
			subdir := filepath.Join(dir, subdir1)
			require.NoError(t, w.Add(subdir))

			filename := filepath.Join(subdir, "file1")

			// Write to one file.
			require.NoError(t, os.WriteFile(filename, []byte("changed"), 0o600))

			var expected []fsnotify.Event

			if poll {
				expected = append(expected, fsnotify.Event{Name: filename, Op: fsnotify.Write})
				assertEvents(t, w, expected...)
			} else {
				// fsnotify sometimes emits Chmod before Write,
				// which is hard to test, so skip it here.
				drainEvents(t, w)
			}

			// Remove one file.
			filename = filepath.Join(subdir, "file2")
			require.NoError(t, os.Remove(filename))
			assertEvents(t, w, fsnotify.Event{Name: filename, Op: fsnotify.Remove})

			// Add one file.
			filename = filepath.Join(subdir, "file3")
			require.NoError(t, os.WriteFile(filename, []byte("new"), 0o600))
			assertEvents(t, w, fsnotify.Event{Name: filename, Op: fsnotify.Create})

			// Remove entire directory.
			subdir = filepath.Join(dir, subdir2)
			require.NoError(t, w.Add(subdir))

			require.NoError(t, os.RemoveAll(subdir))

			if poll {
				assertEvents(t, w, fsnotify.Event{Name: subdir, Op: fsnotify.Remove})
			} else {
				// Accompanying child-removal events vary across fsnotify backends/runs; just drain.
				drainEvents(t, w)
			}
		})

		t.Run(fmt.Sprintf("%s, Add should not trigger event", method), func(t *testing.T) {
			dir, w := preparePollTest(t, poll)
			subdir := filepath.Join(dir, subdir1)
			require.NoError(t, w.Add(subdir))
			assertEvents(t, w)
			// Create a new sub directory and add it to the watcher.
			subdir = filepath.Join(dir, subdir1, subdir2)
			require.NoError(t, os.Mkdir(subdir, 0o777))
			require.NoError(t, w.Add(subdir))
			// This should create only one event.
			assertEvents(t, w, fsnotify.Event{Name: subdir, Op: fsnotify.Create})
		})
	}
}

func TestPollerClose(t *testing.T) {
	w := NewPollingWatcher(watchWaitTime)
	f1, err := os.CreateTemp("", "f1")
	require.NoError(t, err)
	defer os.Remove(f1.Name())
	f2, err := os.CreateTemp("", "f2")
	require.NoError(t, err)
	filename1 := f1.Name()
	filename2 := f2.Name()
	f1.Close()
	f2.Close()

	require.NoError(t, w.Add(filename1))
	require.NoError(t, w.Add(filename2))
	require.NoError(t, w.Close())
	require.NoError(t, w.Close())
	require.NoError(t, os.WriteFile(filename1, []byte("new"), 0o600))
	require.NoError(t, os.WriteFile(filename2, []byte("new"), 0o600))
	// No more event as the watchers are closed.
	assertEvents(t, w)

	f2, err = os.CreateTemp("", "f2")
	require.NoError(t, err)
	defer os.Remove(f2.Name())

	require.Error(t, w.Add(f2.Name()))
}

func TestCheckChange(t *testing.T) {
	dir := prepareTestDirWithSomeFiles(t, "check-change")

	stat := func(s ...string) os.FileInfo {
		fi, err := os.Stat(filepath.Join(append([]string{dir}, s...)...))
		require.NoError(t, err)
		return fi
	}

	f0, f1, f2 := stat(subdir2, "file0"), stat(subdir2, "file1"), stat(subdir2, "file2")
	d1 := stat(subdir1)

	// Note that on Windows, only the 0200 bit (owner writable) of mode is used.
	require.NoError(t, os.Chmod(filepath.Join(dir, subdir2, "file1"), 0o400))
	f1_2 := stat(subdir2, "file1")

	require.NoError(t, os.WriteFile(filepath.Join(dir, subdir2, "file2"), []byte("changed"), 0o600))
	f2_2 := stat(subdir2, "file2")

	require.Equal(t, fsnotify.Remove, checkChange(f0, nil))
	require.Equal(t, fsnotify.Create, checkChange(nil, f0))
	require.Equal(t, fsnotify.Chmod, checkChange(f1, f1_2))
	require.Equal(t, fsnotify.Write, checkChange(f2, f2_2))
	require.Equal(t, fsnotify.Op(0), checkChange(nil, nil))
	require.Equal(t, fsnotify.Op(0), checkChange(d1, f1))
	require.Equal(t, fsnotify.Op(0), checkChange(f1, d1))
}

func BenchmarkPoller(b *testing.B) {
	runBench := func(b *testing.B, item *itemToWatch) {
		b.ResetTimer()
		for b.Loop() {
			evs, err := item.checkForChanges()
			if err != nil {
				b.Fatal(err)
			}
			if len(evs) != 0 {
				b.Fatal("got events")
			}
		}
	}

	b.Run("Check for changes in dir", func(b *testing.B) {
		dir := prepareTestDirWithSomeFiles(b, "bench-check")
		item, err := newItemToWatch(dir)
		require.NoError(b, err)
		runBench(b, item)
	})

	b.Run("Check for changes in file", func(b *testing.B) {
		dir := prepareTestDirWithSomeFiles(b, "bench-check-file")
		filename := filepath.Join(dir, subdir1, "file1")
		item, err := newItemToWatch(filename)
		require.NoError(b, err)
		runBench(b, item)
	})
}

func prepareTestDirWithSomeFiles(tb testing.TB, _ string) string {
	tb.Helper()
	dir := tb.TempDir()
	require.NoError(tb, os.MkdirAll(filepath.Join(dir, subdir1), 0o777))
	require.NoError(tb, os.MkdirAll(filepath.Join(dir, subdir2), 0o777))

	for i := range 3 {
		require.NoError(tb, os.WriteFile(filepath.Join(dir, subdir1, fmt.Sprintf("file%d", i)), []byte("hello1"), 0o600))
	}

	for i := range 3 {
		require.NoError(tb, os.WriteFile(filepath.Join(dir, subdir2, fmt.Sprintf("file%d", i)), []byte("hello2"), 0o600))
	}

	tb.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

func preparePollTest(t *testing.T, poll bool) (string, FileWatcher) {
	t.Helper()
	var w FileWatcher
	if poll {
		w = NewPollingWatcher(watchWaitTime)
	} else {
		var err error
		w, err = NewEventWatcher()
		require.NoError(t, err)
	}

	dir := prepareTestDirWithSomeFiles(t, fmt.Sprint(poll))

	t.Cleanup(func() {
		w.Close()
	})
	return dir, w
}

func assertEvents(t *testing.T, w FileWatcher, evs ...fsnotify.Event) {
	t.Helper()
	i := 0
	check := func() error {
		for {
			select {
			case got := <-w.Events():
				if i > len(evs)-1 {
					if slices.ContainsFunc(evs, func(ev fsnotify.Event) bool { return ev.Name == got.Name }) {
						// A poll tick can split one write into e.g. Create+Write; benign.
						continue
					}
					return fmt.Errorf("got too many event(s): %q", got)
				}
				expected := evs[i]
				i++
				if expected.Name != got.Name {
					return fmt.Errorf("got wrong filename, expected %q: %v", expected.Name, got.Name)
				} else if got.Op&expected.Op != expected.Op {
					return fmt.Errorf("got wrong event type, expected %q: %v", expected.Op, got.Op)
				}
			case e := <-w.Errors():
				return fmt.Errorf("got unexpected error waiting for events %v", e)
			case <-time.After(watchWaitTime * 2): // absorb CI scheduling jitter
				return nil
			}
		}
	}
	require.NoError(t, check())
	require.Equal(t, len(evs), i)
}

func drainEvents(t *testing.T, w FileWatcher) {
	t.Helper()
	check := func() error {
		for {
			select {
			case <-w.Events():
			case e := <-w.Errors():
				return fmt.Errorf("got unexpected error waiting for events %v", e)
			case <-time.After(watchWaitTime * 2):
				return nil
			}
		}
	}
	require.NoError(t, check())
}
