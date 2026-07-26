package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntrypointUnmarshalTOML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   interface{}
		want    entrypoint
		wantErr string
	}{
		{
			name:  "nil clears the entrypoint",
			value: nil,
			want:  nil,
		},
		{
			name:  "string becomes a single element",
			value: "./tmp/main",
			want:  entrypoint{"./tmp/main"},
		},
		{
			name:  "array keeps binary and args",
			value: []interface{}{"./tmp/main", "server", ":8080"},
			want:  entrypoint{"./tmp/main", "server", ":8080"},
		},
		{
			name:  "empty array yields an empty entrypoint",
			value: []interface{}{},
			want:  entrypoint{},
		},
		{
			name:    "non-string array element is rejected",
			value:   []interface{}{"./tmp/main", 42},
			wantErr: "entrypoint values must be strings, got int",
		},
		{
			name:    "unsupported type is rejected",
			value:   42,
			wantErr: "entrypoint must be a string or array of strings, got int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Start from a non-empty value so nil is seen to clear it.
			e := entrypoint{"stale"}
			err := e.UnmarshalTOML(tt.value)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, e)
		})
	}
}

func TestEntrypointBinaryAndArgs(t *testing.T) {
	t.Parallel()

	var empty entrypoint
	assert.Empty(t, empty.binary())
	assert.Nil(t, empty.args())

	single := entrypoint{"./tmp/main"}
	assert.Equal(t, "./tmp/main", single.binary())
	assert.Nil(t, single.args())

	withArgs := entrypoint{"./tmp/main", "server", ":8080"}
	assert.Equal(t, "./tmp/main", withArgs.binary())
	assert.Equal(t, []string{"server", ":8080"}, withArgs.args())
}
