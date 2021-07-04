package runner

import (
	"flag"
)

func CreateArgsFlags() map[string]TomlInfo {
	c := config{}
	m := CreateStructureFieldTagMap(c)
	for k, v := range m {
		flag.StringVar(v.Value, k, "", "")
	}
	return m
}
