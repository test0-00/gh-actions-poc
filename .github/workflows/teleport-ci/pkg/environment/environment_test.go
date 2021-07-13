package environment

import (
	"testing"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/require"
)

func TestNewEnvironment(t *testing.T) {
	tests := []struct {
		cfg      Config
		checkErr require.ErrorAssertionFunc
		expected *Environment
		desc     string
	}{
		{
			cfg: Config{
				Reviewers: `{"foo": ["bar", "baz"]}`,
				Client:    github.NewClient(nil),
			},
			checkErr: require.Error,
			desc:     "invalid Environment config with no token",
			expected: nil,
		},
		{
			cfg: Config{
				Reviewers: `{"foo": ["bar", "baz"]}`,
				Token:     "123456",
				Client:    github.NewClient(nil),
			},
			checkErr: require.NoError,
			desc:     "valid Environment config",
			expected: &Environment{
				Secrets: Secrets{token: "123456", reviewers: map[string][]string{"foo": {"bar", "baz"}}},
				Client:  github.NewClient(nil),
			},
		},
		{
			cfg: Config{
				Reviewers: `{"foo": "baz"}`,
				Token:     "123456",
				Client:    github.NewClient(nil),
			},
			checkErr: require.Error,
			desc:     "invalid assigners object format",
			expected: nil,
		},
		{
			cfg: Config{
				Reviewers: `{"foo": ["baz"]}`,
				Token:     "123456",
			},
			checkErr: require.Error,
			desc:     "invalid config with no client",
			expected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			_, err := New(test.cfg)
			test.checkErr(t, err)
		})
	}
}

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
			res, err := unmarshalReviewers(test.obj)
			test.checkErr(t, err)
			require.EqualValues(t, test.expected, res)
		})
	}

}
