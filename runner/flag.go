package runner

import (
	"flag"
)

const unsetDefault = "DEFAULT"

// ParseConfigFlag parse toml information for flag and register
// keys as Vars in `flag` to be filled later when using `.Parse()`
func ParseConfigFlag(f *flag.FlagSet) map[string]TomlInfo {
	c := Config{}
	m := flatConfig(c)
	for k, v := range m {
		f.StringVar(v.Value, k, unsetDefault, "")
	}
	return m
}
