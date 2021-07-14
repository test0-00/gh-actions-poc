package check

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
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
	teamMembersFn teamMembersFn
	invalidate    invalidate
}

type getPRNumber func(string, string, string, *github.Client) (int, error)
type teamMembersFn func(string, string, *github.Client) ([]string, error)
type invalidate func(string, string, int, map[string]review, *github.Client) error

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
	ch.teamMembersFn = getTeamMembers
	ch.invalidate = invalidateApprovals
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

// check checks to see if all the required reviewers have approved and invalidates 
// approvals for external contributors if a new commit is pushed
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
	ok, err := c.isInternal(c.reviewContext.userLogin)
	if err != nil {
		return trace.Wrap(err)
	}
	if !ok {
		// If all required reviewers reviewed, check if commit shas are all the same
		if hasNewCommit(currentReviews) {
			err := c.invalidate(c.reviewContext.repoOwner, c.reviewContext.repoName, c.reviewContext.number, currentReviews, c.Environment.Client)
			if err != nil {
				return trace.Wrap(err)
			}
			log.Print("invalidating approvals for external contributor.")
			return trace.BadParameter("all required reviewers have not yet approved.")
		}
	}
	return nil
}

func invalidateApprovals(repoOwner, repoName string, number int, reviews map[string]review, clt *github.Client) error {
	for _, v := range reviews {
		_, _, err := clt.PullRequests.DismissReview(context.TODO(), repoOwner, repoName, number, v.id, &github.PullRequestReviewDismissalRequest{})
		if err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// hasNewCommit sees if the pull request has a new commit
// by comparing commits after the push event
func hasNewCommit(revs map[string]review) bool {
	var reviews []review
	if len(revs) == 1 {
		// TODO: if a PR has 1 commit, check if it is the most recent 
		return false
	}
	for _, v := range revs {
		reviews = append(reviews, v)
	}
	i := 0
	for i < len(reviews)-1 {
		if reviews[i].commitID != reviews[i+1].commitID {
			return true
		}
		i++
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
	return c.setReviewContext(body, getPullRequestNumber)
}

// ReviewContext is the pull request review metadata
type ReviewContext struct {
	userLogin string
	repoName  string
	repoOwner string
	number    int
}

// setReviewContext extracts data from body and returns a new instance of pull request review
func (c *Check) setReviewContext(body []byte, fn getPRNumber) error {
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
		clt := c.Environment.Client
		prNumber, err := fn(push.Repository.Owner.Name, push.Repository.Name, push.After, clt)
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

// getPullRequestNumber gets the pull request number associated with a commit sha
// this is used for `push` events because the event payload does not include the pull request number
// as `push` events can occur without a pull request.
func getPullRequestNumber(owner, repo, commitSha string, clt *github.Client) (int, error) {
	pulls, _, err := clt.PullRequests.ListPullRequestsWithCommit(context.TODO(), owner, repo, commitSha, nil)
	if err != nil {
		return -1, err
	}
	switch len(pulls) {
	case 0:
		return -1, trace.NotFound("pull request not found.")

	case 1:
		return -1, trace.BadParameter("ambiguous pull request, cannot determine number.")
	}
	return *pulls[0].Number, nil
}

// isInternal determines if an author is an internal contributor
func (c *Check) isInternal(author string) (bool, error) {
	members, err := c.teamMembersFn(c.Environment.Org, c.Environment.TeamSlug, c.Environment.Client)
	if err != nil {
		return false, trace.Wrap(err)
	}
	if !contains(members, author) {
		return false, nil
	}
	revs := c.Environment.GetReviewersForUser(author)
	if revs == nil {
		return false, nil
	}
	return true, nil
}

func contains(slice []string, value string) bool {
	for i := range slice {
		if slice[i] == value {
			return true
		}
	}
	return false
}

// getTeamMembers gets team members
func getTeamMembers(organization, teamSlug string, client *github.Client) ([]string, error) {
	var teamMembers []string
	members, _, err := client.Teams.ListTeamMembersBySlug(context.TODO(), organization, teamSlug, &github.TeamListTeamMembersOptions{})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	for _, member := range members {
		teamMembers = append(teamMembers, *member.Login)
	}
	return teamMembers, nil
}
