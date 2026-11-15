package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRules(t *testing.T) {
	root := t.TempDir()

	t.Run("defaults and dir normalization", func(t *testing.T) {
		b := cfgBuild{Rules: []cfgRule{{Cmd: "npm run build", IncludeDir: []string{"web"}, ExcludeRegex: []string{`\.map$`}}}}
		require.NoError(t, b.normalizeRules(root))
		rule := b.Rules[0]
		assert.Equal(t, "rule-0", rule.Name)
		assert.Equal(t, []string{filepath.Join(root, "web")}, rule.includeDirAbs)
		assert.Len(t, rule.regexCompiled, 1)
		assert.Equal(t, 1000*time.Millisecond, rule.delay())
	})

	t.Run("cmd is required", func(t *testing.T) {
		b := cfgBuild{Rules: []cfgRule{{Name: "assets", IncludeDir: []string{"web"}}}}
		err := b.normalizeRules(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cmd is required")
	})

	t.Run("at least one matcher is required", func(t *testing.T) {
		b := cfgBuild{Rules: []cfgRule{{Cmd: "true"}}}
		err := b.normalizeRules(root)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one of")
	})

	t.Run("invalid regex", func(t *testing.T) {
		b := cfgBuild{Rules: []cfgRule{{Cmd: "true", ExcludeRegex: []string{"("}}}}
		require.Error(t, b.normalizeRules(root))
	})
}

func TestRuleTomlParsing(t *testing.T) {
	config := `
[build]
cmd = "go build -o ./tmp/main ."

[[build.rules]]
name = "assets"
include_dir = ["web"]
include_ext = ["js", "css"]
cmd = "npm run build"
delay = 200
`
	path := filepath.Join(t.TempDir(), ".air.toml")
	require.NoError(t, os.WriteFile(path, []byte(config), 0o644))

	cfg, err := readConfig(path)
	require.NoError(t, err)
	require.Len(t, cfg.Build.Rules, 1)
	rule := cfg.Build.Rules[0]
	assert.Equal(t, "assets", rule.Name)
	assert.Equal(t, "npm run build", rule.Cmd)
	assert.Equal(t, []string{"web"}, rule.IncludeDir)
	assert.Equal(t, []string{"js", "css"}, rule.IncludeExt)
	assert.Equal(t, 200*time.Millisecond, rule.delay())
}

func TestRuleMatches(t *testing.T) {
	root := t.TempDir()
	cfg := defaultConfig()
	cfg.Root = root
	cfg.Build.Rules = []cfgRule{{
		Name:         "assets",
		Cmd:          "true",
		IncludeDir:   []string{"web"},
		IncludeExt:   []string{"js"},
		IncludeFile:  []string{"web/vite.config"},
		ExcludeRegex: []string{`\.min\.js$`},
	}}
	require.NoError(t, cfg.Build.normalizeRules(root))
	e := &Engine{config: &cfg}

	tests := []struct {
		path  string
		match bool
	}{
		{filepath.Join(root, "web", "app.js"), true},
		{filepath.Join(root, "web", "deep", "nested.js"), true},
		{filepath.Join(root, "web", "app.min.js"), false}, // exclude_regex
		{filepath.Join(root, "web", "index.html"), false}, // ext not matched
		{filepath.Join(root, "web", "vite.config"), true}, // include_file
		{filepath.Join(root, "cmd", "app.js"), false},     // outside include_dir
		{filepath.Join(root, "main.go"), false},
	}
	for _, tt := range tests {
		idx := e.matchRuleIndex(tt.path)
		assert.Equal(t, tt.match, idx >= 0, "path: %s", tt.path)
	}

	assert.True(t, e.isRuleDir(filepath.Join(root, "web")))
	assert.False(t, e.isRuleDir(filepath.Join(root, "web", "deep")))
	assert.True(t, e.inRuleDir(filepath.Join(root, "web", "deep")))
	assert.False(t, e.inRuleDir(filepath.Join(root, "cmd")))
}

// TestRuleRunsCmdWithoutRebuild verifies the core behavior of issue #540: a
// change in a rule's directory runs the rule cmd and does not rebuild the app,
// even when that directory is listed in the main build's exclude_dir.
func TestRuleRunsCmdWithoutRebuild(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires sh")
	}

	tmpDir := t.TempDir()
	t.Setenv(airWd, tmpDir)
	chdir(t, tmpDir)

	config := `
[build]
cmd = "echo built >> builds.txt"
full_bin = "true" # exits immediately
include_ext = ["go"]
exclude_dir = ["web"]

[[build.rules]]
name = "assets"
include_dir = ["web"]
include_ext = ["js"]
cmd = "echo asset >> asset_builds.txt"
delay = 100
`
	require.NoError(t, os.WriteFile(dftTOML, []byte(config), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "web"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "web", "app.js"), []byte("a"), 0o644))

	engine, err := NewEngine(dftTOML, nil, false)
	require.NoError(t, err)
	go engine.Run()
	defer engine.Stop()

	time.Sleep(time.Second)

	countLines := func(name string) int {
		bytes, err := os.ReadFile(name)
		if err != nil {
			return 0
		}
		return len(strings.Split(strings.TrimSpace(string(bytes)), "\n"))
	}

	// first run builds the app once, the rule cmd has not run
	assert.Equal(t, 1, countLines("builds.txt"))
	assert.Equal(t, 0, countLines("asset_builds.txt"))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "web", "app.js"), []byte("b"), 0o644))

	time.Sleep(2 * time.Second)

	assert.Equal(t, 1, countLines("asset_builds.txt"), "rule cmd should have run once")
	assert.Equal(t, 1, countLines("builds.txt"), "changing a rule file must not rebuild the app")
}
