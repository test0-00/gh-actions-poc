package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalReviewers(t *testing.T) {
	tests := []struct {
		obj      string
		expected map[string][]string
		checkErr require.ErrorAssertionFunc
		desc     string
	}{
		{
			obj:      "",
			expected: nil,
			checkErr: require.Error,
			desc:     "empty object",
		},
		{
			obj: `{"foo":["bar"]}`,
			expected: map[string][]string{
				"foo": {"bar"},
			},
			checkErr: require.NoError,
			desc:     "valid user",
		},
		{
			obj:      `{"bar":"foo"}`,
			expected: nil,
			checkErr: require.Error,
			desc:     "invalid object format",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {

			res, err := UnmarshalReviewers(test.obj)
			test.checkErr(t, err)
			require.EqualValues(t, test.expected, res)
		})
	}

}
