package runner

import (
	"flag"
)

const unsetDefault = "DEFAULT"

// ParseConfigFlag parse toml information for flag
func ParseConfigFlag(f *flag.FlagSet) map[string]TomlInfo {
	c := Config{}
	m := FlatConfig(c)
	for k, v := range m {
		f.StringVar(v.Value, k, unsetDefault, "")
	}
	return m
}
