package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeFile creates path (including parents) with the given contents.
func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
}

func TestEngineCacheFileChecksums(t *testing.T) {
	tmpDir := t.TempDir()

	writeFile(t, filepath.Join(tmpDir, "main.go"), "package main\n")
	writeFile(t, filepath.Join(tmpDir, "internal", "app.go"), "package internal\n")
	writeFile(t, filepath.Join(tmpDir, "README.md"), "docs\n")           // extension not watched
	writeFile(t, filepath.Join(tmpDir, "skip.go"), "package skip\n")     // exclude_file
	writeFile(t, filepath.Join(tmpDir, "app_test.go"), "package main\n") // exclude_regex
	writeFile(t, filepath.Join(tmpDir, "tmp", "main.go"), "package tmp\n")
	writeFile(t, filepath.Join(tmpDir, "vendor", "dep.go"), "package dep\n")
	writeFile(t, filepath.Join(tmpDir, ".hidden", "hidden.go"), "package hidden\n")

	cfg := defaultConfig()
	cfg.Root = tmpDir
	cfg.TmpDir = "tmp"
	cfg.Build.IncludeExt = []string{"go"}
	cfg.Build.ExcludeDir = []string{"vendor"}
	cfg.Build.ExcludeFile = []string{"skip.go"}
	cfg.Build.ExcludeRegex = []string{"_test.go"}
	require.NoError(t, cfg.preprocess(nil))

	engine, err := NewEngineWithConfig(&cfg, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.watcher.Close() })

	require.NoError(t, engine.cacheFileChecksums(cfg.Root))

	cached := func(rel string) bool {
		path := filepath.Join(cfg.Root, rel)
		checksum, err := fileChecksum(path)
		require.NoError(t, err)
		// updateFileChecksum reports false when the checksum is already stored.
		return !engine.fileChecksums.updateFileChecksum(path, checksum)
	}

	assert.True(t, cached("main.go"), "watched file should be cached")
	assert.True(t, cached(filepath.Join("internal", "app.go")), "watched file in subdir should be cached")

	assert.False(t, cached("README.md"), "unwatched extension should not be cached")
	assert.False(t, cached("skip.go"), "exclude_file match should not be cached")
	assert.False(t, cached("app_test.go"), "exclude_regex match should not be cached")
	assert.False(t, cached(filepath.Join("vendor", "dep.go")), "exclude_dir content should not be cached")
	assert.False(t, cached(filepath.Join(".hidden", "hidden.go")), "hidden dir content should not be cached")
}

func TestEngineCacheFileChecksumsMissingRoot(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig()
	cfg.Root = tmpDir
	require.NoError(t, cfg.preprocess(nil))

	engine, err := NewEngineWithConfig(&cfg, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.watcher.Close() })

	assert.Error(t, engine.cacheFileChecksums(filepath.Join(tmpDir, "does-not-exist")))
}

func TestEnginePrimeChecksumIgnoresUnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig()
	cfg.Root = tmpDir
	require.NoError(t, cfg.preprocess(nil))

	engine, err := NewEngineWithConfig(&cfg, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.watcher.Close() })

	missing := filepath.Join(tmpDir, "gone.go")
	engine.primeChecksum(missing)

	// Nothing was stored, so the first real update reports a change.
	writeFile(t, missing, "package main\n")
	checksum, err := fileChecksum(missing)
	require.NoError(t, err)
	assert.True(t, engine.fileChecksums.updateFileChecksum(missing, checksum))
}

func TestEngineIsExcludeFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaultConfig()
	cfg.Root = tmpDir
	// isExcludeFile matches with filepath.Match, so the directory pattern has
	// to use the platform separator to match on Windows too.
	cfg.Build.ExcludeFile = []string{"skip.go", filepath.Join("docs", "*.md")}
	require.NoError(t, cfg.preprocess(nil))

	engine, err := NewEngineWithConfig(&cfg, false)
	require.NoError(t, err)
	t.Cleanup(func() { _ = engine.watcher.Close() })

	assert.True(t, engine.isExcludeFile(filepath.Join(cfg.Root, "skip.go")))
	assert.True(t, engine.isExcludeFile(filepath.Join(cfg.Root, "docs", "readme.md")))
	assert.False(t, engine.isExcludeFile(filepath.Join(cfg.Root, "main.go")))
	assert.False(t, engine.isExcludeFile(filepath.Join(cfg.Root, "docs", "nested", "readme.md")))
}
