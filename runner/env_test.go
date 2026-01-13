package runner

import (
	"os"
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

# Inline comments should be stripped
WITH_COMMENT=actual_value # this is a comment
ANOTHER=test#notcomment
QUOTED_HASH="value with # hash"
QUOTED_WITH_COMMENT="quoted value" # trailing comment
SINGLE_QUOTED_COMMENT='single quoted' # also a comment

# Self-referencing
SELF=${SELF}_again

# Expansion
X1=foo
X2=${X1}_bar
X3=${X2}_baz
`

	envPath := t.TempDir() + "/.env"
	if err := os.WriteFile(envPath, []byte(fileContents), 0o644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	unset := []string{
		"FOO", "BAZ", "QUOTED1", "QUOTED2", "UNQUOTED", "BASED",
		"WITH_EQUALS", "SPACY_1", "TRIMMED", "EMPTY1", "EMPTY2",
		"WITH_COMMENT", "ANOTHER", "QUOTED_HASH",
		"QUOTED_WITH_COMMENT", "SINGLE_QUOTED_COMMENT",
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

	if _, err := loadEnvFile(file); err != nil {
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
		{"BASED", "base_123"},
		{"INCOMPLETE", ""},
		{"WITH_EQUALS", "host=localhost;port=5432"},
		{"SPACY_1", "value with spaces"},
		{"TRIMMED", "trimmedVal"},
		{"EMPTY1", ""},
		{"EMPTY2", ""},
		{"WITH_COMMENT", "actual_value"},           // inline comment stripped
		{"ANOTHER", "test#notcomment"},             // no space before #, not a comment
		{"QUOTED_HASH", "value with # hash"},       // quoted values preserve #
		{"QUOTED_WITH_COMMENT", "quoted value"},    // quoted with trailing comment
		{"SINGLE_QUOTED_COMMENT", "single quoted"}, // single quoted with trailing comment
		{"SELF", "_again"},
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
