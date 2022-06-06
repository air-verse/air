package runner

import (
	"flag"
)

func CreateArgsFlags(f *flag.FlagSet) map[string]TomlInfo {
	c := config{}
	m := flatConfig(c)
	for k, v := range m {
		f.StringVar(v.Value, k, "", "")
	}
	return m
}
