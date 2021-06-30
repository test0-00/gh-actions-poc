package check

import (
	"context"

	ci "../../../teleport-ci"
	"../environment"
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

// Check ...
type Check struct {
	Environment environment.Environment
}

// New returns a new instance of Environment
func New(c Config) (*Check, error) {
	var ch *Check

	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}

	err = ch.Environment.NewPullRequest(c.EventPath)
	if err != nil {
		return &Check{}, trace.Wrap(err)
	}
	reviewers, err := environment.UnmarshalReviewers(c.Reviewers)
	if err != nil {
		return &Check{}, trace.Wrap(err)
	}
	secrets := environment.Secrets{
		Assigners: reviewers,
		Token:     c.Token,
	}
	ch.Environment.Secrets = secrets
	ch.Environment.Client = c.Client
	return ch, nil
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

// Check checks if all the reviewers have approved a pull request
// returns nil if all required reviewers have approved, returns error if not
func (c *Check) Check() error {
	env := c.Environment
	listOpts := github.ListOptions{}
	reviews, _, err := env.Client.PullRequests.ListReviews(context.TODO(), env.PullRequest.Head.Repo.Owner.Name,
		env.PullRequest.Head.Repo.Name,
		env.PullRequest.Number,
		&listOpts)

	if err != nil {
		return trace.Wrap(err)
	}
	var currentReviews map[string]Review
	for _, rev := range reviews {
		currentReviews[*rev.User.Name] = Review{Name: *rev.User.Name, Status: *rev.State}
	}
	err = c.check(currentReviews)
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// check checks to see if all the required reviwers are in the current
// reviewer slice
func (c *Check) check(currentReviewers map[string]Review) error {
	// TODO: check if all required reviewers are in current
	required, ok := c.Environment.Secrets.Assigners[c.Environment.PullRequest.User.Login]
	if !ok {
		return trace.BadParameter("user does not exist in map or is an external contributor")
	}
	for _, requiredReviewer := range required {
		rev, ok := currentReviewers[requiredReviewer]
		if !ok {
			return trace.BadParameter("failed to assign all reviewers")
		}
		if rev.Status != ci.APPROVED {
			return trace.BadParameter("all required reviewers have not yet approved")
		}
	}
	return nil
}

// Review is a pull request review
type Review struct {
	Name   string
	Status string
}
