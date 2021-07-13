package check

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"

	ci "github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"

	"github.com/google/go-github/v37/github"
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
	err = ch.SetReviewContext(c.EventPath)
	if err != nil {
		return nil, trace.Wrap(err)
	}
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
		currentReviews[*rev.User.Login] = review{name: *rev.User.Login, status: *rev.State, commitID: *rev.CommitID, id: *rev.ID}
	}
	return c.check(currentReviews)
}

// review is a pull request review
type review struct {
	name     string
	status   string
	commitID string
	id       int64
}

// check checks to see if all the required reviewers have approved
func (c *Check) check(currentReviews map[string]review) error {
	if len(currentReviews) == 0 {
		return trace.BadParameter("pull request has no reviews.")
	}
	required := c.Environment.GetReviewersForUser(c.reviewContext.userLogin)
	for _, requiredReviewer := range required {
		rev, ok := currentReviews[requiredReviewer]
		if !ok {
			return trace.BadParameter("failed to assign all required reviewers.")
		}
		if rev.status != ci.APPROVED {
			return trace.BadParameter("all required reviewers have not yet approved.")
		}
	}
	// If all required reviewers have approved, check if author is external
	if !isInternal(c.reviewContext.userLogin) {
		// If all required reviewers reviewed, check if commit shas are all the same
		if hasNewCommit(currentReviews) {
			err := c.invalidateApprovals(currentReviews)
			if err != nil {
				return trace.Wrap(err)
			}
			return trace.BadParameter("invalidating approvals for external contributor, new commit pushed.")
		}
	}
	return nil
}

func (c *Check) invalidateApprovals(reviews map[string]review) error {
	client := c.Environment.Client
	for _, v := range reviews {
		_, _, err := client.PullRequests.DismissReview(context.TODO(), c.reviewContext.repoOwner,
			c.reviewContext.repoName,
			c.reviewContext.number,
			v.id,
			&github.PullRequestReviewDismissalRequest{})
		if err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// isInternal checks if author is internal
func isInternal(author string) bool {
	return false
}

// hasNewCommit checks to see if all the hashes are the same
func hasNewCommit(revs map[string]review) bool {
	var reviews []review
	for _, v := range revs {
		reviews = append(reviews, v)
	}

	i := 0
	for i < len(reviews)-1 {
		if reviews[i].commitID != reviews[i+1].commitID {
			return true
		}
	}
	return false
}

// SetReviewContext sets reviewContext for Check
func (c *Check) SetReviewContext(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return trace.Wrap(err)
	}
	return c.setReviewContext(body)
}

// newReview extracts data from body and returns a new instance of pull request review
func (c *Check) setReviewContext(body []byte) error {
	// Used on review events
	var rev environment.ReviewMetadata
	err := json.Unmarshal(body, &rev)
	if err != nil {
		return trace.Wrap(err)
	}
	if rev.PullRequest.Number != 0 && rev.Review.User.Login != "" && rev.Repository.Name != "" && rev.Repository.Owner.Name != "" {
		c.reviewContext = &ReviewContext{
			userLogin: rev.Review.User.Login,
			repoName:  rev.Repository.Name,
			repoOwner: rev.Repository.Owner.Name,
			number:    rev.PullRequest.Number,
		}
		return nil
	}

	// Used on push events
	var push environment.PushMetadata
	err = json.Unmarshal(body, &push)
	if err != nil {
		return trace.Wrap(err)
	}
	if push.Pusher.Name != "" && push.Repository.Name != "" && push.Repository.Owner.Name != "" && push.After != "" {
		// Get pull request number
		prNumber, err := c.getPullRequestNumber(push.Repository.Owner.Name, push.Repository.Name, push.After)
		if err != nil {
			return trace.Wrap(err)
		}
		c.reviewContext = &ReviewContext{
			userLogin: push.Pusher.Name,
			repoName:  push.Repository.Name,
			repoOwner: push.Repository.Owner.Name,
			number:    prNumber,
		}
	}
	return trace.BadParameter("insufficient data obtained.")
}

// https://docs.github.com/en/rest/reference/repos#list-pull-requests-associated-with-a-commit
func (c *Check) getPullRequestNumber(owner, repo, commitSha string) (int, error) {
	client := c.Environment.Client
	pulls, _, err := client.PullRequests.ListPullRequestsWithCommit(context.TODO(), owner, repo, commitSha, nil)
	if err != nil {
		return -1, err
	}
	if len(pulls) != 1 {
		return -1, trace.NotFound("pull request not found.")
	}
	return *pulls[0].Number, nil
}

// ReviewContext is the pull request review metadata
type ReviewContext struct {
	userLogin string
	repoName  string
	repoOwner string
	number    int
}
