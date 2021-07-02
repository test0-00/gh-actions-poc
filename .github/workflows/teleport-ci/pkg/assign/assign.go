package assign

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"../environment"

	"github.com/google/go-github/github"
	"github.com/gravitational/trace"
)

// Config ...
type Config struct {
	EventPath   string
	Token       string
	Reviewers   string
	Environment *environment.Environment
}

// Assign ...
type Assign struct {
	Environment *environment.Environment
	pullContext PullRequestContext
}

// CheckAndSetDefaults verifies configuration and sets defaults
func (c *Config) CheckAndSetDefaults() error {
	if c.Environment == nil {
		return trace.BadParameter("missing parameter EventPath or is empty string")
	}
	if c.EventPath == "" {
		return trace.BadParameter("missing parameter EventPath or is empty string")
	}
	return nil
}

// New returns a new instance of Environment
func New(c Config) (*Assign, error) {
	var a Assign
	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	pullContext, err := NewPullRequestContext(c.EventPath)
	if err != nil {
		return &Assign{}, trace.Wrap(err)
	}
	a.pullContext = pullContext
	a.Environment = c.Environment

	return &a, nil
}

// Assign assigns reviewers to the pull request
func (e *Assign) Assign() error {
	r, err := e.Environment.GetReviewersForUser(e.pullContext.userLogin)
	if err != nil {
		return trace.Wrap(err)
	}
	e.Environment.ReviewersRequest = github.ReviewersRequest{Reviewers: r}
	cl := e.Environment.Client
	pr, _, err := cl.PullRequests.RequestReviewers(context.TODO(),
		e.pullContext.repoOwner,
		e.pullContext.repoName, e.pullContext.number,
		e.Environment.ReviewersRequest)
	if err != nil {
		return trace.Wrap(err)
	}

	var reqs map[string]environment.User
	for _, reviewer := range pr.RequestedReviewers {
		reqs[*reviewer.Login] = environment.User{Login: *reviewer.Login}
	}
	return e.assign(reqs)
}

// assign gets the reviewers for the current user event
func (e *Assign) assign(currentReviewers map[string]environment.User) error {
	required, ok := e.Environment.Secrets.Assigners[e.pullContext.userLogin]
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

type review struct {
	reviewer string
	status   string
}

// NewPullRequestContext ...
func NewPullRequestContext(path string) (PullRequestContext, error) {
	file, err := os.Open(path)
	if err != nil {
		return PullRequestContext{}, trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return PullRequestContext{}, trace.Wrap(err)
	}
	return newPullRequestContext(body)
}

func newPullRequestContext(body []byte) (PullRequestContext, error) {
	var pr environment.PRMetadata
	err := json.Unmarshal(body, &pr)
	if err != nil {
		return PullRequestContext{}, trace.Wrap(err)
	}
	if pr.Number == 0 || pr.PullRequest.User.Login == "" || pr.Repository.Name == "" || pr.Repository.Owner.Name == "" {
		return PullRequestContext{}, trace.BadParameter("insufficient data obatined")
	}
	return PullRequestContext{
		number:    pr.Number,
		userLogin: pr.PullRequest.User.Login,
		repoName:  pr.Repository.Name,
		repoOwner: pr.Repository.Owner.Name,
	}, nil
}

// PullRequestContext ...
type PullRequestContext struct {
	number    int
	userLogin string
	repoName  string
	repoOwner string
}
