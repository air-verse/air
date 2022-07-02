package runner

import (
	"flag"
)

const UnsetDefault = "DEFAULT"

// ParseConfigFlag parse toml information for flag
func ParseConfigFlag(f *flag.FlagSet) map[string]TomlInfo {
	c := Config{}
	m := flatConfig(c)
	for k, v := range m {
		f.StringVar(v.Value, k, UnsetDefault, "")
	}
	return m
}
