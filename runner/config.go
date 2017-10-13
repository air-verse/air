package runner

import (
	"io/ioutil"
	"time"

	"github.com/pelletier/go-toml"
)

type config struct {
	Root    string   `toml:"root"`
	TmpPath string   `toml:"tmp_path"`
	Build   cfgBuild `toml:"build"`
	Color   cfgColor `toml:"color"`
}

type cfgBuild struct {
	Bin        string   `toml:"bin"`
	Cmd        string   `toml:"cmd"`
	Log        string   `toml:"log"`
	IncludeExt []string `toml:"include_ext"`
	ExcludeDir []string `toml:"exclude_dir"`
	Delay      int      `toml:"delay"`
}

type cfgColor struct {
	Main    string `toml:"main"`
	Build   string `toml:"build"`
	Runner  string `toml:"runner"`
	Watcher string `toml:"watcher"`
	App     string `toml:"app"`
}

// InitConfig loads config info
func InitConfig(path string) (*config, error) {
	if path == "" {
		dft := defaultConfig()
		return &dft, nil
	}
	// TODO: merge default config
	return readConfig(path)
}

func defaultConfig() config {
	build := cfgBuild{
		Bin:        "./main",
		Cmd:        "go build -o ./main ./main.go",
		Log:        "build-errors.log",
		IncludeExt: []string{"go", "tpl", "tmpl", "html"},
		ExcludeDir: []string{"assets", "tmp", "vendor"},
		Delay:      1000,
	}
	color := cfgColor{
		Main:    "cyan",
		Build:   "yellow",
		Runner:  "green",
		Watcher: "magenta",
		App:     "white",
	}
	return config{
		Root:    ".",
		TmpPath: "./tmp",
		Build:   build,
		Color:   color,
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

func (c *config) colorInfo() map[string]string {
	return map[string]string{
		"main":    c.Color.Main,
		"build":   c.Color.Build,
		"runner":  c.Color.Runner,
		"watcher": c.Color.Watcher,
		"app":     c.Color.App,
	}
}

func (c *config) BuildDelay() time.Duration {
	return time.Duration(c.Build.Delay) * time.Millisecond
}
