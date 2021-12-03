package runner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"

	"github.com/imdario/mergo"
	"github.com/pelletier/go-toml"
)

const (
	dftTOML = ".air.toml"
	dftConf = ".air.conf"
	airWd   = "air_wd"
)

type config struct {
	Root        string    `toml:"root"`
	TmpDir      string    `toml:"tmp_dir"`
	TestDataDir string    `toml:"testdata_dir"`
	Build       cfgBuild  `toml:"build"`
	Color       cfgColor  `toml:"color"`
	Log         cfgLog    `toml:"log"`
	Misc        cfgMisc   `toml:"misc"`
	Screen      cfgScreen `toml:"screen"`
}

type cfgBuild struct {
	Cmd              string        `toml:"cmd"`
	Bin              string        `toml:"bin"`
	FullBin          string        `toml:"full_bin"`
	Log              string        `toml:"log"`
	IncludeExt       []string      `toml:"include_ext"`
	ExcludeDir       []string      `toml:"exclude_dir"`
	IncludeDir       []string      `toml:"include_dir"`
	ExcludeFile      []string      `toml:"exclude_file"`
	ExcludeRegex     []string      `toml:"exclude_regex"`
	ExcludeUnchanged bool          `toml:"exclude_unchanged"`
	FollowSymlink    bool          `toml:"follow_symlink"`
	Delay            int           `toml:"delay"`
	StopOnError      bool          `toml:"stop_on_error"`
	SendInterrupt    bool          `toml:"send_interrupt"`
	KillDelay        time.Duration `toml:"kill_delay"`
	regexCompiled    []*regexp.Regexp
}

func (c *cfgBuild) RegexCompiled() ([]*regexp.Regexp, error) {
	if len(c.ExcludeRegex) > 0 && len(c.regexCompiled) == 0 {
		c.regexCompiled = make([]*regexp.Regexp, 0, len(c.ExcludeRegex))
		for _, s := range c.ExcludeRegex {
			re, err := regexp.Compile(s)
			if err != nil {
				return nil, err
			}
			c.regexCompiled = append(c.regexCompiled, re)
		}
	}
	return c.regexCompiled, nil
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

type cfgScreen struct {
	ClearOnRebuild bool `toml:"clear_on_rebuild"`
}

func initConfig(path string) (cfg *config, err error) {
	if path == "" {
		cfg, err = defaultPathConfig()
		if err != nil {
			return nil, err
		}
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

func writeDefaultConfig() {
	confFiles := []string{dftTOML, dftConf}

	for _, fname := range confFiles {
		fstat, err := os.Stat(fname)
		if err != nil && !os.IsNotExist(err) {
			log.Fatal("failed to check for existing configuration")
			return
		}
		if err == nil && fstat != nil {
			log.Fatal("configuration already exists")
			return
		}
	}

	file, err := os.Create(dftTOML)
	if err != nil {
		log.Fatalf("failed to create a new confiuration: %+v", err)
	}
	defer file.Close()

	config := defaultConfig()
	configFile, err := toml.Marshal(config)
	if err != nil {
		log.Fatalf("failed to marshal the default configuration: %+v", err)
	}

	_, err = file.Write(configFile)
	if err != nil {
		log.Fatalf("failed to write to %s: %+v", dftTOML, err)
	}

	fmt.Printf("%s file created to the current directory with the default settings\n", dftTOML)
}

func defaultPathConfig() (*config, error) {
	// when path is blank, first find `.air.toml`, `.air.conf` in `air_wd` and current working directory, if not found, use defaults
	for _, name := range []string{dftTOML, dftConf} {
		cfg, err := readConfByName(name)
		if err == nil {
			if name == dftConf {
				fmt.Println("`.air.conf` will be deprecated soon, recommend using `.air.toml`.")
			}
			return cfg, nil
		}
	}

	dftCfg := defaultConfig()
	return &dftCfg, nil
}

func readConfByName(name string) (*config, error) {
	var path string
	if wd := os.Getenv(airWd); wd != "" {
		path = filepath.Join(wd, name)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(wd, name)
	}
	cfg, err := readConfig(path)
	return cfg, err
}

func defaultConfig() config {
	build := cfgBuild{
		Cmd:          "go build -o ./tmp/main .",
		Bin:          "./tmp/main",
		Log:          "build-errors.log",
		IncludeExt:   []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir:   []string{"assets", "tmp", "vendor", "testdata"},
		ExcludeRegex: []string{"_test.go"},
		Delay:        1000,
		StopOnError:  true,
	}
	if runtime.GOOS == PlatformWindows {
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
		Root:        ".",
		TmpDir:      "tmp",
		TestDataDir: "testdata",
		Build:       build,
		Color:       color,
		Log:         log,
		Misc:        misc,
	}
}

func readConfig(path string) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := new(config)
	if err = toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func readConfigOrDefault(path string) (*config, error) {
	dftCfg := defaultConfig()
	cfg, err := readConfig(path)
	if err != nil {
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
	if c.TestDataDir == "" {
		c.TestDataDir = "testdata"
	}
	if err != nil {
		return err
	}
	ed := c.Build.ExcludeDir
	for i := range ed {
		ed[i] = cleanPath(ed[i])
	}

	adaptToVariousPlatforms(c)

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

func (c *config) buildLogPath() string {
	return filepath.Join(c.tmpPath(), c.Build.Log)
}

func (c *config) buildDelay() time.Duration {
	return time.Duration(c.Build.Delay) * time.Millisecond
}

func (c *config) binPath() string {
	return filepath.Join(c.Root, c.Build.Bin)
}

func (c *config) tmpPath() string {
	return filepath.Join(c.Root, c.TmpDir)
}

func (c *config) TestDataPath() string {
	return filepath.Join(c.Root, c.TestDataDir)
}

func (c *config) rel(path string) string {
	s, err := filepath.Rel(c.Root, path)
	if err != nil {
		return ""
	}
	return s
}
