package runner

import (
	"flag"
	"reflect"
)

// appendableValue is a flag.Value for slice-typed config fields. The first
// occurrence replaces the default, every later occurrence appends, so
// `-env_files a,b -env_files c` is equivalent to `-env_files a,b,c`.
type appendableValue struct {
	p   *string
	set bool
}

func (v *appendableValue) String() string {
	if v == nil || v.p == nil {
		return ""
	}
	return *v.p
}

func (v *appendableValue) Set(s string) error {
	switch {
	case !v.set:
		*v.p = s
	case s == "":
		// nothing to append
	case *v.p == "":
		*v.p = s
	default:
		*v.p += sliceCmdArgSeparator + s
	}
	v.set = true
	return nil
}

// ParseConfigFlag parse toml information for flag
func ParseConfigFlag(f *flag.FlagSet) map[string]TomlInfo {
	c := defaultConfig()
	m := flatConfig(c)
	for k, v := range m {
		if v.field.Type.Kind() == reflect.Slice {
			*v.Value = v.fieldValue
			f.Var(&appendableValue{p: v.Value}, k, v.usage)
			continue
		}
		f.StringVar(v.Value, k, v.fieldValue, v.usage)
	}
	return m
}
