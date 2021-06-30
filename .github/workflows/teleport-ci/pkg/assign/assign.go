package assign

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/google/go-github/github"
	"github.com/gravitational/trace"
)

// Config ...
type Config struct {
	Client    *github.Client
	EventPath string
	Token     string
	Reviewers string
}

// CheckAndSetDefaults verifies configuration and sets defaults
func (c *Config) CheckAndSetDefaults() error {
	if c.Client == nil {
		return trace.BadParameter("missing parameter Client")
	}
	if c.EventPath == "" {
		return trace.BadParameter("missing parameter EventPath or is empty string")
	}
	if c.Token == "" {
		return trace.BadParameter("missing parameter EventPath or is empty string")
	}
	if c.Reviewers == "" {
		return trace.BadParameter("missing parameter Reviewers")
	}
	return nil
}

// Environment is the environment metadata
type Environment struct {
	PullRequest PullRequest `json:"pull_request"`
	Secrets     Secrets
	Client      *github.Client
}

// New returns a new instance of Environment
func New(c Config) (*Environment, error) {
	var env *Environment

	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = env.NewPullRequest(c.EventPath)
	if err != nil {
		return &Environment{}, trace.Wrap(err)
	}
	reviewers, err := unmarshalReviewers(c.Reviewers)
	if err != nil {
		return &Environment{}, trace.Wrap(err)
	}
	secrets := Secrets{
		Assigners: reviewers,
		Token:     c.Token,
	}
	env.Secrets = secrets
	env.Client = c.Client
	return env, nil
}

// Assign assigns reviewers to the pull request
func (e *Environment) Assign() error {
	r, err := e.getReviewersForUser()
	if err != nil {
		return trace.Wrap(err)
	}
	reviewers := github.ReviewersRequest{Reviewers: r}
	pr, _, err := e.Client.PullRequests.RequestReviewers(context.TODO(),
		e.PullRequest.Head.Repo.Owner.Name,
		e.PullRequest.Head.Repo.Name, e.PullRequest.Number,
		reviewers)
	if err != nil {
		return trace.Wrap(err)
	}

	var reqs map[string]User
	for _, reviewer := range pr.RequestedReviewers {
		reqs[*reviewer.Login] = User{*reviewer.Login}
	}
	return e.assign(reqs)
}

// assign gets the reviewers for the current user event
func (e *Environment) assign(currentReviewers map[string]User) error {
	required, ok := e.Secrets.Assigners[e.PullRequest.User.Login]
	if !ok {
		return trace.BadParameter("user does not exist in map or is an external contributor")
	}
	for _, requiredReviewer := range required {
		if _, ok := currentReviewers[requiredReviewer]; !ok {
			return trace.BadParameter("failed to assign all reviewers")
		}
	}
	return nil
}

// getReviewersForUser gets the required reviewers for the current user
func (e *Environment) getReviewersForUser() ([]string, error) {
	value, ok := e.Secrets.Assigners[e.PullRequest.User.Login]
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

// NewPullRequest unmarshals pull request metadata from json file given the path
func (e *Environment) NewPullRequest(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return trace.Wrap(err)
	}
	return e.newPullRequest(body)
}

func (e *Environment) newPullRequest(body []byte) error {
	err := json.Unmarshal(body, e)
	if err != nil {
		return trace.Wrap(err)
	}
	pullReq := e.PullRequest
	if pullReq.Number == 0 || pullReq.User.Login == "" || pullReq.Head.Repo.Name == "" || pullReq.Head.Repo.Owner.Name == "" {
		return trace.BadParameter("insufficient data obatined")
	}
	return nil
}

// PullRequest ...
type PullRequest struct {
	Number int `json:"number"`
	User   User
	Head   Head `json:"head"`
}

// User ...
type User struct {
	Login string `json:"login"`
}

// Head ...
type Head struct {
	Repo Repo `json:"repo"`
}

// Repo ...
type Repo struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

// Owner ...
type Owner struct {
	Name string `json:"login"`
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
