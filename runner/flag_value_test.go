package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendableValueString(t *testing.T) {
	t.Parallel()

	var nilValue *appendableValue
	assert.Empty(t, nilValue.String(), "nil receiver should stringify to empty")

	assert.Empty(t, (&appendableValue{}).String(), "nil target should stringify to empty")

	s := "a,b"
	assert.Equal(t, "a,b", (&appendableValue{p: &s}).String())
}

func TestAppendableValueSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		initial string
		sets    []string
		want    string
	}{
		{
			name:    "first set replaces the default",
			initial: "default",
			sets:    []string{"a"},
			want:    "a",
		},
		{
			name:    "later sets append",
			initial: "default",
			sets:    []string{"a", "b,c"},
			want:    "a" + sliceCmdArgSeparator + "b,c",
		},
		{
			name:    "empty append is ignored",
			initial: "default",
			sets:    []string{"a", ""},
			want:    "a",
		},
		{
			name:    "append onto an emptied value does not add a separator",
			initial: "default",
			sets:    []string{"", "b"},
			want:    "b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			target := tt.initial
			v := &appendableValue{p: &target}
			for _, s := range tt.sets {
				require.NoError(t, v.Set(s))
			}
			assert.Equal(t, tt.want, target)
			assert.Equal(t, tt.want, v.String())
		})
	}
}
