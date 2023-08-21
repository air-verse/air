package runner

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"time"

	"dario.cat/mergo"
	"github.com/pelletier/go-toml"
	"gopkg.in/yaml.v3"
)

const (
	dftTOML = ".air.toml"
	dftYAML = ".air.yaml"
	dftYML  = ".air.yml"
	dftConf = ".air.conf"
	airWd   = "air_wd"
)

// Config is the main configuration structure for Air.
type Config struct {
	Root        string    `toml:"root" yaml:"root"`
	TmpDir      string    `toml:"tmp_dir" yaml:"tmp_dir"`
	TestDataDir string    `toml:"testdata_dir" yaml:"testdata_dir"`
	Build       cfgBuild  `toml:"build" yaml:"build"`
	Color       cfgColor  `toml:"color" yaml:"color"`
	Log         cfgLog    `toml:"log" yaml:"log"`
	Misc        cfgMisc   `toml:"misc" yaml:"misc"`
	Screen      cfgScreen `toml:"screen" yaml:"screen"`
}

type cfgBuild struct {
	Cmd              string        `toml:"cmd" yaml:"cmd"`
	Bin              string        `toml:"bin" yaml:"bin"`
	FullBin          string        `toml:"full_bin" yaml:"full_bin"`
	ArgsBin          []string      `toml:"args_bin" yaml:"args_bin"`
	Log              string        `toml:"log" yaml:"log"`
	IncludeExt       []string      `toml:"include_ext" yaml:"include_ext"`
	ExcludeDir       []string      `toml:"exclude_dir" yaml:"exclude_dir"`
	IncludeDir       []string      `toml:"include_dir" yaml:"include_dir"`
	ExcludeFile      []string      `toml:"exclude_file" yaml:"exclude_file"`
	IncludeFile      []string      `toml:"include_file" yaml:"include_file"`
	ExcludeRegex     []string      `toml:"exclude_regex" yaml:"exclude_regex"`
	ExcludeUnchanged bool          `toml:"exclude_unchanged" yaml:"exclude_unchanged"`
	FollowSymlink    bool          `toml:"follow_symlink" yaml:"follow_symlink"`
	Poll             bool          `toml:"poll" yaml:"poll"`
	PollInterval     int           `toml:"poll_interval" yaml:"poll_interval"`
	Delay            int           `toml:"delay" yaml:"delay"`
	StopOnError      bool          `toml:"stop_on_error" yaml:"stop_on_error"`
	SendInterrupt    bool          `toml:"send_interrupt" yaml:"send_interrupt"`
	KillDelay        time.Duration `toml:"kill_delay" yaml:"kill_delay"`
	Rerun            bool          `toml:"rerun" yaml:"rerun"`
	RerunDelay       int           `toml:"rerun_delay" yaml:"rerun_delay"`
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
	AddTime  bool `toml:"time" yaml:"time"`
	MainOnly bool `toml:"main_only" yaml:"main_only"`
}

type cfgColor struct {
	Main    string `toml:"main" yaml:"main"`
	Watcher string `toml:"watcher" yaml:"watcher"`
	Build   string `toml:"build" yaml:"build"`
	Runner  string `toml:"runner" yaml:"runner"`
	App     string `toml:"app" yaml:"app"`
}

type cfgMisc struct {
	CleanOnExit bool `toml:"clean_on_exit" yaml:"clean_on_exit"`
}

type cfgScreen struct {
	ClearOnRebuild bool `toml:"clear_on_rebuild" yaml:"clear_on_rebuild"`
	KeepScroll     bool `toml:"keep_scroll" yaml:"keep_scroll"`
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
func InitConfig(path string) (cfg *Config, name string, err error) {
	name = path
	if path == "" {
		cfg, name, err = defaultPathConfig()
		if err != nil {
			return nil, name, err
		}
	} else {
		cfg, err = readConfigOrDefault(path)
		if err != nil {
			return nil, name, err
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
		return nil, name, err
	}

	err = ret.preprocess()
	return ret, name, err
}

func writeDefaultConfig() {
	confFiles := []string{dftTOML, dftYAML, dftYML, dftConf}

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
		log.Fatalf("failed to create a new configuration: %+v", err)
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

func defaultPathConfig() (*Config, string, error) {
	// when path is blank, first find `.air.toml`, `.air.yaml`, `.air.yml`,`.air.conf` in `air_wd` and current working directory, if not found, use defaults
	for _, name := range []string{dftTOML, dftYAML, dftYML, dftConf} {
		cfg, err := readConfByName(name)
		if err == nil {
			if name == dftConf {
				fmt.Println("`.air.conf` will be deprecated soon, recommend using `.air.toml`.")
			}
			return cfg, name, nil
		}
	}

	dftCfg := defaultConfig()
	return &dftCfg, "", nil
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
		ExcludeFile:  []string{},
		IncludeFile:  []string{},
		ExcludeDir:   []string{"assets", "tmp", "vendor", "testdata"},
		ArgsBin:      []string{},
		ExcludeRegex: []string{"_test.go"},
		Delay:        0,
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
	ext := filepath.Ext(path)
	switch ext {
	case ".yml", ".yaml":
		if err = yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}

	default:
		if err = toml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
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
