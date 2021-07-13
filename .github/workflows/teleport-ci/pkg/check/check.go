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
	reviewContext *ReviewContext
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
		return trace.BadParameter("missing parameter Environment.")
	}
	if c.EventPath == "" {
		return trace.BadParameter("missing parameter EventPath.")
	}
	return nil
}

// Check checks if all the reviewers have approved a pull request
// returns nil if all required reviewers have approved or returns an error if not
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

	currentReviews := make(map[string]review)
	for _, rev := range reviews {
		currentReviews[*rev.User.Login] = review{name: *rev.User.Login, status: *rev.State}
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

// check checks to see if all the required reviewers have approved
func (c *Check) check(currentReviews map[string]review) error {
	if len(currentReviews) == 0 {
		return trace.BadParameter("pull request has no reviews.")
	}
	required, err := c.Environment.GetReviewersForUser(c.reviewContext.userLogin)
	if err != nil {
		return trace.Wrap(err)
	}

	for _, requiredReviewer := range required {
		rev, ok := currentReviews[requiredReviewer]
		if !ok {
			return trace.BadParameter("failed to assign all required reviewers.")
		}
		if rev.status != ci.APPROVED {
			return trace.BadParameter("all required reviewers have not yet approved.")
		}
	}
	return nil
}

// NewReviewContext unmarshals pull request review metadata from json file given the path
func NewReviewContext(path string) (*ReviewContext, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	return newReviewContext(body)
}

// newReview extracts data from body and returns a new instance of pull request review
func newReviewContext(body []byte) (*ReviewContext, error) {
	var rev environment.ReviewMetadata
	err := json.Unmarshal(body, &rev)
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if rev.PullRequest.Number == 0 || rev.Review.User.Login == "" || rev.Repository.Name == "" || rev.Repository.Owner.Name == "" {
		return nil, trace.BadParameter("insufficient data obtained.")
	}
	return &ReviewContext{
		userLogin: rev.Review.User.Login,
		repoName:  rev.Repository.Name,
		repoOwner: rev.Repository.Owner.Name,
		number:    rev.PullRequest.Number,
	}, nil
}

// ReviewContext is the pull request review metadata
type ReviewContext struct {
	userLogin string
	repoName  string
	repoOwner string
	number    int
}
