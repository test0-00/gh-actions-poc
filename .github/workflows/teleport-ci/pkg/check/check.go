package check

import (
	"context"
	"encoding/json"
	"fmt"
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
	action        string
}

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
	log.Printf("current reviews: %+v", currentReviews)
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
	required := c.Environment.GetReviewersForAuthor(c.reviewContext.author)
	log.Printf("checking if %v has approvals from the required reviewers %+v", c.reviewContext.author, required)

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
	ok, err := c.isInternal(c.reviewContext.author)
	if err != nil {
		return trace.Wrap(err)
	}
	if !ok {
		// If all required reviewers reviewed, check if commit shas are all the same
		if c.hasNewCommit(currentReviews) {
			err := c.invalidate(c.reviewContext.repoOwner, c.reviewContext.repoName, c.reviewContext.number, currentReviews, c.Environment.Client)
			if err != nil {
				return trace.Wrap(err)
			}
			log.Printf("invalidating approvals for external contributor, %v", c.reviewContext.author)
			return trace.BadParameter("all required reviewers have not yet approved.")
		}
	}
	return nil
}

func invalidateApprovals(repoOwner, repoName string, number int, reviews map[string]review, clt *github.Client) error {
	msg := fmt.Sprint("bot.")
	for _, v := range reviews {
		_, _, err := clt.PullRequests.DismissReview(context.TODO(), repoOwner, repoName, number, v.id, &github.PullRequestReviewDismissalRequest{Message: &msg})
		if err != nil {
			return trace.Wrap(err)
		}
	}
	return nil
}

// hasNewCommit sees if the pull request has a new commit
// by comparing commits after the push event
func (c *Check) hasNewCommit(revs map[string]review) bool {
	for _, v := range revs {
		if v.commitID != c.reviewContext.headSHA {
			return true
		}
	}
	return false
}

type action struct {
	Action string `json:"action"`
}

// SetReviewContext sets reviewContext for Check
func (c *Check) SetReviewContext(path string) error {
	var actionType action
	file, err := os.Open(path)
	if err != nil {
		return trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return trace.Wrap(err)
	}
	err = json.Unmarshal(body, &actionType)
	if err != nil {
		return trace.Wrap(err)
	}
	c.action = actionType.Action
	return c.setReviewContext(body)
}

// ReviewContext is the pull request review metadata
type ReviewContext struct {
	author string
	// Only used for pull request review events
	reviewer  string
	repoName  string
	repoOwner string
	number    int
	headSHA   string
}

// setReviewContext extracts data from body and returns a new instance of pull request review
func (c *Check) setReviewContext(body []byte) error {
	switch c.action {
	case "synchronize":
		// Used on push events
		var push environment.PRMetadata
		err := json.Unmarshal(body, &push)
		if err != nil {
			return trace.Wrap(err)
		}
		if push.Number != 0 && push.Repository.Name != "" && push.Repository.Owner.Name != "" && push.PullRequest.User.Login != "" && push.CommitSHA != "" {
			c.reviewContext = &ReviewContext{
				author:    push.PullRequest.User.Login,
				repoName:  push.Repository.Name,
				repoOwner: push.Repository.Owner.Name,
				number:    push.Number,
				headSHA:   push.CommitSHA,
			}
			return nil
		}
	default:
		// Used on review events
		var rev environment.ReviewMetadata
		err := json.Unmarshal(body, &rev)
		if err != nil {
			return trace.Wrap(err)
		}

		if rev.PullRequest.Number != 0 && rev.Review.User.Login != "" && rev.Repository.Name != "" && rev.Repository.Owner.Name != "" {
			c.reviewContext = &ReviewContext{
				author:    rev.PullRequest.Author.Login,
				reviewer:  rev.Review.User.Login,
				repoName:  rev.Repository.Name,
				repoOwner: rev.Repository.Owner.Name,
				number:    rev.PullRequest.Number,
				headSHA:   rev.PullRequest.Head.SHA,
			}
			return nil
		}
	}
	return trace.BadParameter("insufficient data obtained.")
}

// isInternal determines if an author is an internal contributor
func (c *Check) isInternal(author string) (bool, error) {
	members, err := c.teamMembersFn(c.Environment.Org, c.Environment.TeamSlug, c.Environment.Client)
	if err != nil {
		log.Printf("failed to evaluate if author %v is part of %v: %v", c.reviewContext.author, c.Environment.TeamSlug, err)
		return false, nil
	}
	if !contains(members, author) {
		return false, nil
	}
	revs := c.Environment.GetReviewersForAuthor(author)
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
