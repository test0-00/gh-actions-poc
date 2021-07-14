package environment

import (
	"testing"

	"github.com/google/go-github/v37/github"
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
				Reviewers:        `{"foo": ["bar", "baz"]}`,
				Client:           github.NewClient(nil),
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin"},
			},
			checkErr: require.Error,
			desc:     "invalid Environment config with no token",
			expected: nil,
		},
		{
			cfg: Config{
				Client:           github.NewClient(nil),
				Token:            "1234",
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin"},
				Reviewers:        `{"foo": ["bar", "baz"]}`,
			},
			checkErr: require.NoError,
			desc:     "valid Environment config",
			expected: &Environment{
				token:     "123456",
				reviewers: map[string][]string{"foo": {"bar", "baz"}},
				Client:    github.NewClient(nil),
			},
		},
		{
			cfg: Config{
				Reviewers:        `{"foo": "baz"}`,
				Token:            "123456",
				Client:           github.NewClient(nil),
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin"},
			},
			checkErr: require.Error,
			desc:     "invalid assigners object format",
			expected: nil,
		},
		{
			cfg:      Config{},
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
