package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	t.Setenv("BASE_ENV", "base")

	const fileContents = `
# Comment line at the top

FOO=bar
BAZ=qux

# Quoted values
QUOTED1="hello world"
QUOTED2='foo bar'
UNQUOTED=no quotes

# Expansion from existing env
BASED=${BASE_ENV}_123
NESTED1=${QUOTED1}_suffix

# Expansion from other keys in the file in dependency order
CHILD=${PARENT}/kid
PARENT=${GRANDPARENT}/parent
GRANDPARENT=grand

# Out-of-order recursive dependency
DEEP3=${DEEP2}/c
DEEP2=${DEEP1}/b
DEEP1=deepbase

# Invalid lines (ignore)
justsomegarbage
=badkey
NOEQUALS
INCOMPLETE=

# Equals sign in value
WITH_EQUALS=host=localhost;port=5432

# Whitespace variations
   SPACY_1   =    value with spaces    
TRIMMED =trimmedVal

# Empty lines and values

EMPTY1=
EMPTY2 =   

# More comments
# final marker

# Self-referencing
SELF=${SELF}_again

# Complex chained expansion
X1=foo
X2=${X1}_bar
X3=${X2}_baz
`

	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envPath, []byte(fileContents), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	unset := []string{
		"FOO", "BAZ", "QUOTED1", "QUOTED2", "UNQUOTED", "BASED", "NESTED1",
		"CHILD", "PARENT", "GRANDPARENT", "DEEP3", "DEEP2", "DEEP1",
		"WITH_EQUALS", "SPACY_1", "TRIMMED", "EMPTY1", "EMPTY2",
		"SELF", "X1", "X2", "X3",
	}
	for _, k := range unset {
		os.Unsetenv(k)
	}

	file, err := os.Open(envPath)
	if err != nil {
		t.Fatalf("failed to open env file: %v", err)
	}
	defer file.Close()

	if err := loadEnvFile(file); err != nil {
		t.Fatalf("loadEnvFile failed: %v", err)
	}

	cases := []struct {
		key, want string
	}{
		{"FOO", "bar"},
		{"BAZ", "qux"},
		{"QUOTED1", "hello world"},
		{"QUOTED2", "foo bar"},
		{"UNQUOTED", "no quotes"},
		{"BASED", "base_123"}, // must pick up pre existing env var BASE_ENV
		{"NESTED1", "hello world_suffix"},
		{"CHILD", "grand/parent/kid"},
		{"PARENT", "grand/parent"},
		{"GRANDPARENT", "grand"},
		{"DEEP1", "deepbase"},
		{"DEEP2", "deepbase/b"},
		{"DEEP3", "deepbase/b/c"},
		{"WITH_EQUALS", "host=localhost;port=5432"},
		{"SPACY_1", "value with spaces"},
		{"TRIMMED", "trimmedVal"},
		{"EMPTY1", ""},
		{"EMPTY2", ""},
		{"SELF", "_again"}, // Because it starts unset, so ${SELF} expands to ""
		{"X1", "foo"},
		{"X2", "foo_bar"},
		{"X3", "foo_bar_baz"},
	}

	for _, c := range cases {
		if got := os.Getenv(c.key); got != c.want {
			t.Errorf("%s = %q, want %q", c.key, got, c.want)
		}
	}

	// Ensure ignored or invalid lines did not create variables
	invalid := []string{"justsomegarbage", "badkey", "NOEQUALS", "INCOMPLETE"}
	for _, key := range invalid {
		if v := os.Getenv(key); v != "" {
			t.Errorf("unexpected var %q: got %q, want unset", key, v)
		}
	}
}

func TestEnvFileConfig(t *testing.T) {
	t.Parallel()

	t.Run("default env_file is .env", func(t *testing.T) {
		t.Parallel()
		cfg := defaultConfig()
		if cfg.EnvFile != ".env" {
			t.Errorf("EnvFile = %q, want %q", cfg.EnvFile, ".env")
		}
	})
}
