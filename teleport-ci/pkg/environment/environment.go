package environment

import (
	"encoding/json"

	"github.com/google/go-github/github"
	"github.com/gravitational/trace"
)

// Config is used to configure Environment
type Config struct {
	Client    *github.Client
	Token     string
	Reviewers string
}

// Environment contains information about the environment 
type Environment struct {
	Secrets   Secrets
	Client    *github.Client
	ReviewersRequest github.ReviewersRequest
}

// CheckAndSetDefaults verifies configuration and sets defaults
func (c *Config) CheckAndSetDefaults() error {
	if c.Client == nil {
		return trace.BadParameter("missing parameter Client")
	}
	if c.Token == "" {
		return trace.BadParameter("missing parameter EventPath or is empty string")
	}
	if c.Reviewers == "" {
		return trace.BadParameter("missing parameter Reviewers")
	}
	return nil
}

// New ...
func New(c Config) (*Environment, error) {
	var env Environment

	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	reviewers, err := UnmarshalReviewers(c.Reviewers)
	if err != nil {
		return &Environment{}, trace.Wrap(err)
	}
	env.Secrets.Assigners = reviewers
	env.Secrets.Token = c.Token
	env.Client = c.Client

	return &env, nil
}

// GetReviewersForUser gets the required reviewers for the current user
func (e *Environment) GetReviewersForUser(user string) ([]string, error) {
	value, ok := e.Secrets.Assigners[user]
	if !ok {
		return nil, trace.BadParameter("author not found")
	}
	return value, nil
}

// UnmarshalReviewers ...
func UnmarshalReviewers(str string) (map[string][]string, error) {
	if str == "" {
		return nil, trace.BadParameter("reviewers not found")
	}
	m := make(map[string][]string)

	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Secrets ...
type Secrets struct {
	Assigners map[string][]string
	Token     string
}

type review struct {
	reviewer string
	status   string
}

// ReviewMetadata ...
type ReviewMetadata struct {
	Review      Review      `json:"review"`
	Repository  Repository  `json:"repository"`
	PullRequest PullRequest `json:"pull_request"`
}

// Review ...
type Review struct {
	User User `json:"user"`
}

// User ...
type User struct {
	Login string `json:"login"`
}

// PullRequest ...
type PullRequest struct {
	Number int `json:"number"`
}

// PRMetadata ...
type PRMetadata struct {
	Number      int        `json:"number"`
	PullRequest PR         `json:"pull_request"`
	Repository  Repository `json:"repository"`
}

// PR ...
type PR struct {
	User User
}

// Repository ...
type Repository struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

// Owner ...
type Owner struct {
	Name string `json:"login"`
}
