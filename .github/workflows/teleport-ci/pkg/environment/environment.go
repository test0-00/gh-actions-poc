package environment

import (
	"encoding/json"
	"log"

	"github.com/gravitational/trace"

	"github.com/google/go-github/v37/github"
)

// Config is used to configure Environment
type Config struct {
	Client    *github.Client
	Token     string
	Reviewers string
	TeamSlug  string
	Org       string
	// DefaultReviewers is used for external contributors
	DefaultReviewers []string
}

// Environment contains information about the environment
type Environment struct {
	Client           *github.Client
	TeamSlug         string
	Org              string
	reviewers        map[string][]string
	defaultReviewers []string
	token            string
}

// CheckAndSetDefaults verifies configuration and sets defaults
func (c *Config) CheckAndSetDefaults() error {
	if c.Client == nil {
		return trace.BadParameter("missing parameter Client.")
	}
	if c.Token == "" {
		return trace.BadParameter("missing parameter Token.")
	}
	if c.Reviewers == "" {
		return trace.BadParameter("missing parameter Reviewers.")
	}
	if c.TeamSlug == "" {
		return trace.BadParameter("missing parameter TeamSlug.")
	}
	if c.Org == "" {
		return trace.BadParameter("missing parameter Org.")
	}
	if c.DefaultReviewers == nil {
		return trace.BadParameter("missing parameter DefaultReviewers.")
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
	env.Client = c.Client
	revs, err := unmarshalReviewers(c.Reviewers)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	env.reviewers = revs
	env.token = c.Token
	env.TeamSlug = c.TeamSlug
	env.Org = c.Org
	env.defaultReviewers = c.DefaultReviewers
	return &env, nil
}

func unmarshalReviewers(str string) (map[string][]string, error) {
	if str == "" {
		return nil, trace.BadParameter("reviewers not found.")
	}
	m := make(map[string][]string)

	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetReviewersForAuthor gets the required reviewers for the current user
func (e *Environment) GetReviewersForAuthor(user string) []string {
	value, ok := e.reviewers[user]
	// author is external or does not have set reviewers
	if !ok {
		log.Printf("getting default reviewers: %+v", e.defaultReviewers)
		return e.defaultReviewers
	}
	log.Printf("getting reviewers for %+v: %+v", user, value)
	return value
}

func (e *Environment) IsInternal(author string) bool {
	_, ok := e.reviewers[author]
	// author is external or does not have set reviewers
	if !ok {
		return false
	}
	return true
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
	CommitSHA   string     `json:"after"`
}

// ReviewMetadata contains metadata about the pull request
// review (used for the pull request review event)
type ReviewMetadata struct {
	Review      Review      `json:"review"`
	Repository  Repository  `json:"repository"`
	PullRequest PullRequest `json:"pull_request"`
}

// Head contains the commit sha at the head of the pull request
type Head struct {
	SHA string `json:"sha"`
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
	Author User `json:"user"`
	Number int  `json:"number"`
	Head   Head `json:"head"`
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
