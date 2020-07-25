package runner

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"github.com/pelletier/go-toml"
)

const (
	dftConf = ".air.conf"
	airWd   = "air_wd"
)

type config struct {
	Root   string   `toml:"root"`
	TmpDir string   `toml:"tmp_dir"`
	Build  cfgBuild `toml:"build"`
	Color  cfgColor `toml:"color"`
	Log    cfgLog   `toml:"log"`
	Misc   cfgMisc  `toml:"misc"`
}

type cfgBuild struct {
	Cmd           string        `toml:"cmd"`
	Bin           string        `toml:"bin"`
	FullBin       string        `toml:"full_bin"`
	Log           string        `toml:"log"`
	IncludeExt    []string      `toml:"include_ext"`
	ExcludeDir    []string      `toml:"exclude_dir"`
	IncludeDir    []string      `toml:"include_dir"`
	ExcludeFile   []string      `toml:"exclude_file"`
	Delay         int           `toml:"delay"`
	StopOnError   bool          `toml:"stop_on_error"`
	SendInterrupt bool          `toml:"send_interrupt"`
	KillDelay     time.Duration `toml:"kill_delay"`
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

type cfgMisc struct {
	CleanOnExit bool `toml:"clean_on_exit"`
}

func initConfig(path string) (cfg *config, err error) {
	if path == "" {
		// when path is blank, first find `.air.conf` in `air_wd` and current working directory, if not found, use defaults
		if wd := os.Getenv(airWd); wd != "" {
			path = filepath.Join(wd, dftConf)
		} else {
			path, err = dftConfPath()
			if err != nil {
				return nil, err
			}
		}
		cfg, _ = readConfigOrDefault(path)
	} else {
		cfg, err = readConfigOrDefault(path)
		if err != nil {
			return nil, err
		}
	}
	err = mergo.Merge(cfg, defaultConfig())
	if err != nil {
		return nil, err
	}
	err = cfg.preprocess()
	return cfg, err
}

func defaultConfig() config {
	build := cfgBuild{
		Cmd:         "go build -o ./tmp/main .",
		Bin:         "./tmp/main",
		Log:         "build-errors.log",
		IncludeExt:  []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir:  []string{"assets", "tmp", "vendor"},
		Delay:       1000,
		StopOnError: true,
	}
	if runtime.GOOS == "windows" {
		build.Bin = `tmp\main.exe`
		build.Cmd = "go build -o ./tmp/main.exe ."
	}
	log := cfgLog{
		AddTime: false,
	}
	color := cfgColor{
		Main:    "magenta",
		Watcher: "cyan",
		Build:   "yellow",
		Runner:  "green",
	}
	misc := cfgMisc{
		CleanOnExit: false,
	}
	return config{
		Root:   ".",
		TmpDir: "tmp",
		Build:  build,
		Color:  color,
		Log:    log,
		Misc:   misc,
	}
}

func readConfigOrDefault(path string) (*config, error) {
	dftCfg := defaultConfig()
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return &dftCfg, err
	}
	cfg := new(config)
	if err = toml.Unmarshal(data, cfg); err != nil {
		return &dftCfg, err
	}
	return cfg, nil
}

func (c *config) preprocess() error {
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

	// Fix the default configuration is not used in Windows
	// Use the unix configuration on Windows
	if runtime.GOOS == "windows" {

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
				c.Build.FullBin = runName + " " + c.Build.FullBin
			}
		}

		// bin=/tmp/main  cmd=go build -o ./tmp/main.exe main.go
		if !strings.Contains(c.Build.Cmd, c.Build.Bin) && strings.Contains(c.Build.Cmd, originBin) {
			c.Build.Cmd = strings.Replace(c.Build.Cmd, originBin, c.Build.Bin, 1)
		}
	}

	c.Build.ExcludeDir = ed
	if len(c.Build.FullBin) > 0 {
		c.Build.Bin = c.Build.FullBin
		return err
	}
	// Fix windows CMD processor
	// CMD will not recognize relative path like ./tmp/server
	c.Build.Bin, err = filepath.Abs(c.Build.Bin)
	return err
}

func (c *config) colorInfo() map[string]string {
	return map[string]string{
		"main":    c.Color.Main,
		"build":   c.Color.Build,
		"runner":  c.Color.Runner,
		"watcher": c.Color.Watcher,
	}
}

func dftConfPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, dftConf), nil
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
