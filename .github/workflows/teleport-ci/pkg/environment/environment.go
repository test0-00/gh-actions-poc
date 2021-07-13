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
	Secrets Secrets
	Client  *github.Client
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

// New creates a new instance of environment
func New(c Config) (*Environment, error) {
	var env Environment

	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	reviewers, err := unmarshalReviewers(c.Reviewers)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	env.Secrets.reviewers = reviewers
	env.Secrets.token = c.Token
	env.Client = c.Client

	return &env, nil
}

// GetReviewersForUser gets the required reviewers for the current user
func (e *Environment) GetReviewersForUser(user string) ([]string, error) {
	value, ok := e.Secrets.reviewers[user]
	if !ok {
		return nil, trace.BadParameter("author not found")
	}
	return value, nil
}

func unmarshalReviewers(str string) (map[string][]string, error) {
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

/*
   Below are struct definitions used to transform pull request and review
   events (represented as a json object) into Golang structs. The way these objects are
   structured are different, therefore separate structs for each event are needed
   to unmarshal appropiately even though the end result essentially contains
   the same information.
*/

// PRMetadata contains metadata about the pull request (used for the pull request event)
type PRMetadata struct {
	Number      int        `json:"number"`
	PullRequest PR         `json:"pull_request"`
	Repository  Repository `json:"repository"`
}

// ReviewMetadata contains metadata about the pull request
// review (used for the pull request review event)
type ReviewMetadata struct {
	Review      Review      `json:"review"`
	Repository  Repository  `json:"repository"`
	PullRequest PullRequest `json:"pull_request"`
}

// Secrets contains environment secrets
type Secrets struct {
	reviewers map[string][]string
	token     string
}

// Review contains information about the pull request review
type Review struct {
	User User `json:"user"`
}

// User contains information about the user
type User struct {
	Login string `json:"login"`
}

// PullRequest contains information about the pull request (used for pull request review event)
type PullRequest struct {
	Number int `json:"number"`
}

// PR contains information about the pull request (used for the pull request event)
type PR struct {
	User User
}

// Repository contains information about the repository
type Repository struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

// Owner contains information about the repository owner
type Owner struct {
	Name string `json:"login"`
}
