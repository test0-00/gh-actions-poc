package check

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	ci "github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"

	"github.com/google/go-github/github"
	"github.com/gravitational/trace"
)

// Config is used to configure Check
type Config struct {
	EventPath   string
	Token       string
	Reviewers   string
	Environment *environment.Environment
}

// Check checks assigned reviewers for a pull request on a review event
type Check struct {
	Environment   *environment.Environment
	reviewContext ReviewContext
}

// New returns a new instance of  Check
func New(c Config) (*Check, error) {
	var ch Check

	err := c.CheckAndSetDefaults()
	if err != nil {
		return nil, trace.Wrap(err)
	}
	revContext, err := NewReviewContext(c.EventPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	ch.reviewContext = revContext
	ch.Environment = c.Environment

	return &ch, nil
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

// Check checks if all the reviewers have approved a pull request
// returns nil if all required reviewers have approved, returns error if not
func (c *Check) Check() error {
	env := c.Environment
	listOpts := github.ListOptions{}
	reviews, _, err := env.Client.PullRequests.ListReviews(context.TODO(), c.reviewContext.repoOwner,
		c.reviewContext.repoName,
		c.reviewContext.number,
		&listOpts)

	if err != nil {
		return trace.Wrap(err)
	}
	var currentReviews map[string]review
	for _, rev := range reviews {
		currentReviews[*rev.User.Name] = review{name: *rev.User.Name, status: *rev.State}
	}
	err = c.check(currentReviews)
	if err != nil {
		return trace.Wrap(err)
	}
	return nil
}

// Review is a pull request review
type review struct {
	name   string
	status string
}

// check checks to see if all the required reviwers are in the current
// reviewer slice
func (c *Check) check(currentReviews map[string]review) error {
	if len(currentReviews) == 0 {
		return trace.BadParameter("pull request has no reviews")
	}
	required, ok := c.Environment.Secrets.Assigners[c.reviewContext.userLogin]
	if !ok {
		return trace.BadParameter("author is unknown or is an external contributor")
	}
	for _, requiredReviewer := range required {
		rev, ok := currentReviews[requiredReviewer]
		if !ok {
			return trace.BadParameter("failed to assign all required reviewers")
		}
		if rev.status != ci.APPROVED {
			return trace.BadParameter("all required reviewers have not yet approved")
		}
	}
	return nil
}

// NewReviewContext unmarshals pull request metadata from json file given the path
func NewReviewContext(path string) (ReviewContext, error) {
	file, err := os.Open(path)
	if err != nil {
		return ReviewContext{}, trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return ReviewContext{}, trace.Wrap(err)
	}
	return newReviewContext(body)
}

// newReview extracts data from body and returns a new instance of pull request review
func newReviewContext(body []byte) (ReviewContext, error) {
	var rev environment.ReviewMetadata
	err := json.Unmarshal(body, &rev)
	if err != nil {
		return ReviewContext{}, trace.Wrap(err)
	}
	if rev.PullRequest.Number == 0 || rev.Review.User.Login == "" || rev.Repository.Name == "" || rev.Repository.Owner.Name == "" {
		return ReviewContext{}, trace.BadParameter("insufficient data obatined")
	}
	return ReviewContext{
		userLogin: rev.Review.User.Login,
		repoName:  rev.Repository.Name,
		repoOwner: rev.Repository.Owner.Name,
		number:    rev.PullRequest.Number,
	}, nil
}

// ReviewContext ...
type ReviewContext struct {
	userLogin string
	repoName  string
	repoOwner string
	number    int
}
