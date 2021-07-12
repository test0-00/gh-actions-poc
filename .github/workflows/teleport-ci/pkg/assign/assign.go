package assign

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/google/go-github/github"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"
	"github.com/gravitational/trace"
)

// Config is used to configure Assign
type Config struct {
	EventPath   string
	Reviewers   string
	Environment *environment.Environment
}

// Assign assigns reviewers to a pull request on a pull request event
type Assign struct {
	Environment *environment.Environment
	pullContext *PullRequestContext
}

// CheckAndSetDefaults verifies configuration and sets defaults
func (c *Config) CheckAndSetDefaults() error {
	if c.Environment == nil {
		return trace.BadParameter("missing parameter Environment.")
	}
	if c.EventPath == "" {
		return trace.BadParameter("missing parameter EventPath.")
	}
	return nil
}

// New returns a new instance of Assign
func New(c Config) (*Assign, error) {
	var a Assign
	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	pullContext, err := NewPullRequestContext(c.EventPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	a.pullContext = pullContext
	a.Environment = c.Environment

	return &a, nil
}

// Assign assigns reviewers to the pull request
func (e *Assign) Assign() error {
	if e.pullContext == nil {
		return trace.BadParameter("missing pull request data.")
	}
	if e.pullContext.userLogin == "" {
		return trace.BadParameter("current user not found.")
	}
	// Getting and setting reviewers for author of pull request
	r, err := e.Environment.GetReviewersForUser(e.pullContext.userLogin)
	if err != nil {
		return trace.Wrap(err)
	}
	e.Environment.ReviewersRequest = github.ReviewersRequest{Reviewers: r}
	cl := e.Environment.Client
	// Assigning reviewers to pull request
	pr, _, err := cl.PullRequests.RequestReviewers(context.TODO(),
		e.pullContext.repoOwner,
		e.pullContext.repoName, e.pullContext.number,
		e.Environment.ReviewersRequest)
	if err != nil {
		return trace.Wrap(err)
	}

	var reqs map[string]bool
	for _, reviewer := range pr.RequestedReviewers {
		reqs[*reviewer.Login] = true
	}
	return e.assign(reqs)
}

// assign verifies reviewers are assigned
func (e *Assign) assign(currentReviewers map[string]bool) error {
	required, ok := e.Environment.Secrets.Assigners[e.pullContext.userLogin]
	if !ok {
		return trace.BadParameter("user does not exist or is an external contributor.")
	}
	for _, requiredReviewer := range required {
		if !currentReviewers[requiredReviewer] {
			return trace.BadParameter("failed to assign all reviewers.")
		}
	}
	return nil
}

type review struct {
	reviewer string
	status   string
}

// NewPullRequestContext creates a new instance of PullRequestContext
func NewPullRequestContext(path string) (*PullRequestContext, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return newPullRequestContext(body)
}

func newPullRequestContext(body []byte) (*PullRequestContext, error) {
	var pr environment.PRMetadata
	err := json.Unmarshal(body, &pr)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if pr.Number == 0 || pr.PullRequest.User.Login == "" || pr.Repository.Name == "" || pr.Repository.Owner.Name == "" {
		return nil, trace.BadParameter("insufficient data obatined.")
	}
	return &PullRequestContext{
		number:    pr.Number,
		userLogin: pr.PullRequest.User.Login,
		repoName:  pr.Repository.Name,
		repoOwner: pr.Repository.Owner.Name,
	}, nil
}

// PullRequestContext contains information about the pull request event
type PullRequestContext struct {
	number    int
	userLogin string
	repoName  string
	repoOwner string
}
