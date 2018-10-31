package runner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pelletier/go-toml"
)

const (
	dftConf = ".air.conf"
	airWd   = "air_wd"
)

type config struct {
	Root     string   `toml:"root"`
	WatchDir string   `toml:"watch_dir"`
	TmpDir   string   `toml:"tmp_dir"`
	Build    cfgBuild `toml:"build"`
	Color    cfgColor `toml:"color"`
	Log      cfgLog   `toml:"log"`
}

type cfgBuild struct {
	Bin        string   `toml:"bin"`
	Args       []string `toml:"args"`
	Cmd        string   `toml:"cmd"`
	Log        string   `toml:"log"`
	IncludeExt []string `toml:"include_ext"`
	ExcludeDir []string `toml:"exclude_dir"`
	Delay      int      `toml:"delay"`
}

type cfgLog struct {
	AddTime bool `toml:"time"`
}

type cfgColor struct {
	Main    string `toml:"main"`
	Watcher string `toml:"watcher"`
	Build   string `toml:"build"`
	Runner  string `toml:"runner"`
	App     string `toml:"app"`
}

func initConfig(path string) (*config, error) {
	var err error
	var useDftCfg bool
	dft := defaultConfig()
	if path == "" {
		useDftCfg = true
		// when path is blank, first find `.air.conf` in `air_wd` and current working directory, if not found, use defaults
		wd := os.Getenv(airWd)
		if wd != "" {
			path = filepath.Join(wd, dftConf)
		} else {
			path, err = dftConfPath()
			if err != nil {
				return &dft, nil
			}
		}
	}
	cfg, err := readConfig(path)
	if err != nil {
		if !useDftCfg {
			return nil, err
		}
		cfg = &dft
	}
	err = cfg.preprocess()
	return cfg, err
}

func defaultConfig() config {
	build := cfgBuild{
		Bin:        "tmp/main",
		Cmd:        "go build -o ./tmp/main main.go",
		Log:        "build-errors.log",
		IncludeExt: []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir: []string{"assets", "tmp", "vendor"},
		Delay:      1000,
	}
	if runtime.GOOS == "windows" {
		build.Bin = `tmp\main.exe`
		build.Cmd = "go build -o ./tmp/main.exe main.go"
	}
	color := cfgColor{
		Main:    "magenta",
		Watcher: "cyan",
		Build:   "yellow",
		Runner:  "green",
		App:     "white",
	}
	log := cfgLog{
		AddTime: true,
	}
	return config{
		Root:     ".",
		TmpDir:   "tmp",
		WatchDir: "",
		Build:    build,
		Color:    color,
		Log:      log,
	}
}

func readConfig(path string) (*config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := config{}
	if err = toml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *config) preprocess() error {
	// TODO: merge defaults if some options are not set
	var err error
	cwd := os.Getenv(airWd)
	if cwd != "" {
		if err = os.Chdir(cwd); err != nil {
			return err
		}
		c.Root = cwd
	}
	c.Root, err = expandPath(c.Root)
	if c.TmpDir == "" {
		c.TmpDir = "tmp"
	}
	if err != nil {
		return err
	}
	ed := c.Build.ExcludeDir
	for i := range ed {
		ed[i] = cleanPath(ed[i])
	}
	c.Build.ExcludeDir = ed
	return nil
}

func (c *config) colorInfo() map[string]string {
	return map[string]string{
		"main":    c.Color.Main,
		"build":   c.Color.Build,
		"runner":  c.Color.Runner,
		"watcher": c.Color.Watcher,
		"app":     c.Color.App,
	}
}

func dftConfPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, dftConf), nil
}

func (c *config) watchDirRoot() string {
	if c.WatchDir != "" {
		return c.fullPath(c.WatchDir)
	}
	return c.Root
}

func (c *config) buildLogPath() string {
	return filepath.Join(c.tmpPath(), c.Build.Log)
}

func (c *config) buildDelay() time.Duration {
	return time.Duration(c.Build.Delay) * time.Millisecond
}

func (c *config) fullPath(path string) string {
	return filepath.Join(c.Root, path)
}

func (c *config) binPath() string {
	return filepath.Join(c.Root, c.Build.Bin)
}

func (c *config) tmpPath() string {
	return filepath.Join(c.Root, c.TmpDir)
}

func (c *config) rel(path string) string {
	s, err := filepath.Rel(c.Root, path)
	if err != nil {
		return ""
	}
	return s
}
