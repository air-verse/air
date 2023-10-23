package runner

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"time"

	"dario.cat/mergo"
	"github.com/pelletier/go-toml"
)

const (
	dftTOML = ".air.toml"
	dftConf = ".air.conf"
	airWd   = "air_wd"
)

// Config is the main configuration structure for Air.
type Config struct {
	Root        string    `toml:"root" comment:". or absolute path, please note that the directories following must be under root."`
	TmpDir      string    `toml:"tmp_dir" comment:"Temporary directory."`
	TestDataDir string    `toml:"testdata_dir"`
	Build       cfgBuild  `toml:"build"`
	Color       cfgColor  `toml:"color"`
	Log         cfgLog    `toml:"log"`
	Misc        cfgMisc   `toml:"misc"`
	Screen      cfgScreen `toml:"screen"`
}

type cfgBuild struct {
	PreCmd           []string      `toml:"pre_cmd" comment:"Array of commands to run before each build"`
	Cmd              string        `toml:"cmd" comment:"Shell command to run for building."`
	PostCmd          []string      `toml:"post_cmd" comment:"Array of commands to run after ^C"`
	Bin              string        `toml:"bin" comment:"Binary file yields from cmd."`
	FullBin          string        `toml:"full_bin" comment:"Customize binary, can setup environment variables when run your app."`
	ArgsBin          []string      `toml:"args_bin" comment:"Add additional arguments when running binary (bin/full_bin)."`
	Log              string        `toml:"log" comment:"Log file places in your tmp_dir."`
	IncludeExt       []string      `toml:"include_ext" comment:"Watch these filename extensions."`
	ExcludeDir       []string      `toml:"exclude_dir" comment:"Ignore these filename extensions or directories."`
	IncludeDir       []string      `toml:"include_dir" comment:"Watch these directories if you specified."`
	ExcludeFile      []string      `toml:"exclude_file" comment:"Exclude these files."`
	IncludeFile      []string      `toml:"include_file" comment:"Watch these files."`
	ExcludeRegex     []string      `toml:"exclude_regex" comment:"Exclude specific regular expressions."`
	ExcludeUnchanged bool          `toml:"exclude_unchanged" comment:"Exclude unchanged files."`
	FollowSymlink    bool          `toml:"follow_symlink" comment:"Follow symlink for directories"`
	Poll             bool          `toml:"poll" comment:"Poll files for changes instead of using fsnotify."`
	PollInterval     int           `toml:"poll_interval" comment:"Poll interval (defaults to the minimum interval of 500ms)."`
	Delay            int           `toml:"delay" comment:"Delay in milliseconds."`
	StopOnError      bool          `toml:"stop_on_error" comment:"Stop running old binary when build errors occur."`
	SendInterrupt    bool          `toml:"send_interrupt" comment:"Send Interrupt signal before killing process (windows does not support this feature)"`
	KillDelay        time.Duration `toml:"kill_delay" comment:"Delay after sending Interrupt signal in nanoseconds."`
	Rerun            bool          `toml:"rerun" comment:"Rerun binary or not"`
	RerunDelay       int           `toml:"rerun_delay" comment:"Delay after each executions in milliseconds."`
	regexCompiled    []*regexp.Regexp
}

type cfgLog struct {
	AddTime  bool `toml:"time" comment:"Show log time"`
	MainOnly bool `toml:"main_only" comment:"Only show main log (silences watcher, build, runner)"`
}

type cfgColor struct {
	Main    string `toml:"main" comment:"Customize main part's color. If no color found, use the raw app log."`
	Watcher string `toml:"watcher" comment:"Customize watcher part's color."`
	Build   string `toml:"build" comment:"Customize build part's color."`
	Runner  string `toml:"runner" comment:"Customize runner part's color."`
	App     string `toml:"app" comment:"Customize app part's color."`
}

type cfgMisc struct {
	CleanOnExit bool `toml:"clean_on_exit" comment:"Delete tmp directory on exit"`
}

type cfgScreen struct {
	ClearOnRebuild bool `toml:"clear_on_rebuild" comment:"Clear screen on rebuild"`
	KeepScroll     bool `toml:"keep_scroll" comment:"Keep scroll position"`
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

type sliceTransformer struct{}

func (t sliceTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ.Kind() == reflect.Slice {
		return func(dst, src reflect.Value) error {
			if !src.IsZero() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}

// InitConfig initializes the configuration.
func InitConfig(path string) (cfg *Config, err error) {
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
	config := defaultConfig()
	// get addr
	ret := &config
	err = mergo.Merge(ret, cfg, func(config *mergo.Config) {
		// mergo.Merge will overwrite the fields if it is Empty
		// So need use this to avoid that none-zero slice will be overwritten.
		// https://dario.cat/mergo#transformers
		config.Transformers = sliceTransformer{}
		config.Overwrite = true
	})
	if err != nil {
		return nil, err
	}

	err = ret.preprocess()
	return ret, err
}

func writeDefaultConfig() (string, error) {
	confFiles := []string{dftTOML, dftConf}

	for _, fname := range confFiles {
		fstat, err := os.Stat(fname)
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to check for existing configuration: %w", err)
		}
		if err == nil && fstat != nil {
			return "", errors.New("configuration already exists")
		}
	}

	file, err := os.Create(dftTOML)
	if err != nil {
		return "", fmt.Errorf("failed to create a new configuration: %w", err)
	}
	defer file.Close()

	config := defaultConfig()
	configFile, err := toml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the default configuration: %w", err)
	}

	_, err = file.Write(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to write to %s: %w", dftTOML, err)
	}

	return dftTOML, nil
}

func defaultPathConfig() (*Config, error) {
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

func readConfByName(name string) (*Config, error) {
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

func defaultConfig() Config {
	build := cfgBuild{
		Cmd:          "go build -o ./tmp/main .",
		Bin:          "./tmp/main",
		Log:          "build-errors.log",
		IncludeExt:   []string{"go", "tpl", "tmpl", "html"},
		IncludeDir:   []string{},
		PreCmd:       []string{},
		PostCmd:      []string{},
		ExcludeFile:  []string{},
		IncludeFile:  []string{},
		ExcludeDir:   []string{"assets", "tmp", "vendor", "testdata"},
		ArgsBin:      []string{},
		ExcludeRegex: []string{"_test.go"},
		Delay:        1000,
		Rerun:        false,
		RerunDelay:   500,
	}
	if runtime.GOOS == PlatformWindows {
		build.Bin = `tmp\main.exe`
		build.Cmd = "go build -o ./tmp/main.exe ."
	}
	log := cfgLog{
		AddTime:  false,
		MainOnly: false,
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
	return Config{
		Root:        ".",
		TmpDir:      "tmp",
		TestDataDir: "testdata",
		Build:       build,
		Color:       color,
		Log:         log,
		Misc:        misc,
		Screen: cfgScreen{
			ClearOnRebuild: false,
			KeepScroll:     true,
		},
	}
}

func readConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := new(Config)
	if err = toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func readConfigOrDefault(path string) (*Config, error) {
	dftCfg := defaultConfig()
	cfg, err := readConfig(path)
	if err != nil {
		return &dftCfg, err
	}

	return cfg, nil
}

func (c *Config) preprocess() error {
	var err error
	cwd := os.Getenv(airWd)
	if cwd != "" {
		if err = os.Chdir(cwd); err != nil {
			return err
		}
		c.Root = cwd
	}
	c.Root, err = expandPath(c.Root)
	if err != nil {
		return err
	}
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

	// Join runtime arguments with the configuration arguments
	runtimeArgs := flag.Args()
	c.Build.ArgsBin = append(c.Build.ArgsBin, runtimeArgs...)

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

func (c *Config) colorInfo() map[string]string {
	return map[string]string{
		"main":    c.Color.Main,
		"build":   c.Color.Build,
		"runner":  c.Color.Runner,
		"watcher": c.Color.Watcher,
	}
}

func (c *Config) buildLogPath() string {
	return filepath.Join(c.tmpPath(), c.Build.Log)
}

func (c *Config) buildDelay() time.Duration {
	return time.Duration(c.Build.Delay) * time.Millisecond
}

func (c *Config) rerunDelay() time.Duration {
	return time.Duration(c.Build.RerunDelay) * time.Millisecond
}

func (c *Config) binPath() string {
	return filepath.Join(c.Root, c.Build.Bin)
}

func (c *Config) tmpPath() string {
	return filepath.Join(c.Root, c.TmpDir)
}

func (c *Config) testDataPath() string {
	return filepath.Join(c.Root, c.TestDataDir)
}

func (c *Config) rel(path string) string {
	s, err := filepath.Rel(c.Root, path)
	if err != nil {
		return ""
	}
	return s
}

// WithArgs returns a new config with the given arguments added to the configuration.
func (c *Config) WithArgs(args map[string]TomlInfo) {
	for _, value := range args {
		if value.Value != nil && *value.Value != unsetDefault {
			v := reflect.ValueOf(c)
			setValue2Struct(v, value.fieldPath, *value.Value)
		}
	}
}
