package assign

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/google/go-github/v37/github"
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
	err = a.SetPullRequestContext(c.EventPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	a.Environment = c.Environment
	return &a, nil
}

// Assign assigns reviewers to the pull request
func (a *Assign) Assign() error {
	// Getting and setting reviewers for author of pull request
	r := a.Environment.GetReviewersForAuthor(a.pullContext.author)
	client := a.Environment.Client
	// Assigning reviewers to pull request
	pr, _, err := client.PullRequests.RequestReviewers(context.TODO(),
		a.pullContext.repoOwner,
		a.pullContext.repoName, a.pullContext.number,
		github.ReviewersRequest{Reviewers: r})
	if err != nil {
		return trace.Wrap(err)
	}

	reqs := make(map[string]bool)
	for _, reviewer := range pr.RequestedReviewers {
		reqs[*reviewer.Login] = true
	}
	return a.assign(reqs)
}

// assign verifies reviewers are assigned
func (a *Assign) assign(currentReviewers map[string]bool) error {
	required := a.Environment.GetReviewersForAuthor(a.pullContext.author)

	for _, requiredReviewer := range required {
		if !currentReviewers[requiredReviewer] {
			return trace.BadParameter("failed to assign all reviewers.")
		}
	}
	return nil
}

// SetPullRequestContext creates a new instance of PullRequestContext
func (a *Assign) SetPullRequestContext(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return trace.Wrap(err)
	}
	return a.setPullRequestContext(body)
}

func (a *Assign) setPullRequestContext(body []byte) error {
	var pr environment.PRMetadata
	err := json.Unmarshal(body, &pr)
	if err != nil {
		return trace.Wrap(err)
	}
	if pr.Number == 0 || pr.PullRequest.User.Login == "" || pr.Repository.Name == "" || pr.Repository.Owner.Name == "" {
		return trace.BadParameter("insufficient data obtained.")
	}
	a.pullContext = &PullRequestContext{
		number:    pr.Number,
		author: pr.PullRequest.User.Login,
		repoName:  pr.Repository.Name,
		repoOwner: pr.Repository.Owner.Name,
	}
	return nil
}

// PullRequestContext contains information about the pull request event
type PullRequestContext struct {
	number    int
	author string
	repoName  string
	repoOwner string
}
