package runner

import (
	"flag"
)

// ParseConfigFlag parse toml information for flag
func ParseConfigFlag(f *flag.FlagSet) map[string]TomlInfo {
	c := defaultConfig()
	m := flatConfig(c)
	for k, v := range m {
		f.StringVar(v.Value, k, v.fieldValue, v.usage)
	}
	return m
}
