package check

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"

	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
)

func TestNewCheck(t *testing.T) {
	env, err := environment.New(environment.Config{
		Client:           github.NewClient(nil),
		Token:            "12345",
		Reviewers:        `{"foo": ["bar", "baz"]}`,
		TeamSlug:         "team-name",
		Org:              "org",
		DefaultReviewers: []string{},
	})
	require.NoError(t, err)
	// Config with invalid path
	config := Config{
		EventPath:   "path/to/file.json",
		Environment: env,
	}
	assign, err := New(config)
	require.Error(t, err)
	require.Nil(t, assign)

	f, err := ioutil.TempFile("", "check")
	require.NoError(t, err)
	filePath := f.Name()
	defer os.Remove(f.Name())
	_, err = f.Write([]byte(validString))
	require.NoError(t, err)

	// Config with a nil Environment and valid path
	config = Config{
		EventPath: filePath,
	}
	assign, err = New(config)
	require.Error(t, err)
	require.Nil(t, assign)

	// Valid config
	f, err = ioutil.TempFile("", "check")
	require.NoError(t, err)
	filePath = f.Name()
	defer os.Remove(f.Name())
	_, err = f.Write([]byte(validString))
	require.NoError(t, err)
	config = Config{
		EventPath:   filePath,
		Environment: env,
	}
	assign, err = New(config)
	require.NoError(t, err)
	require.Equal(t, env, assign.Environment)

	// Valid config, wrong event (invalid json format)
	f, err = ioutil.TempFile("", "invalid-check")
	require.NoError(t, err)
	filePath = f.Name()
	defer os.Remove(f.Name())
	_, err = f.Write([]byte(invalidString))
	require.NoError(t, err)
	config = Config{
		EventPath:   filePath,
		Environment: env,
	}
	assign, err = New(config)
	require.Error(t, err)
	require.Nil(t, assign)
}

// TestNewReviewContextValid tests the unmarshalling of a valid review event
func TestSetReviewContextValidReviewEvent(t *testing.T) {
	ch := Check{}

	err := ch.setReviewContext([]byte(validString))
	require.NoError(t, err)
	require.Equal(t, 2, ch.reviewContext.number)
	require.Equal(t, "Codertocat", ch.reviewContext.userLogin)
	require.Equal(t, "Hello-World", ch.reviewContext.repoName)
	require.Equal(t, "Codertocat", ch.reviewContext.repoOwner)
}

// TestNewReviewContextInvalid tests the unmarshalling of an event that is not a review (i.e. pull request event)
func TestSetReviewContextInvalidReviewEvent(t *testing.T) {
	ch := Check{}
	err := ch.setReviewContext([]byte(invalidString))
	require.Error(t, err)

	err = ch.setReviewContext([]byte(""))
	require.Error(t, err)

	err = ch.setReviewContext([]byte(invalidStringNoLogin))
	require.Error(t, err)
}

func TestSetReviewContextValidPushEvent(t *testing.T) {
	ch := Check{Environment: &environment.Environment{Client: github.NewClient(nil)}}
	err := ch.setReviewContext([]byte(validString))
	require.NoError(t, err)
	require.Equal(t, 2, ch.reviewContext.number)
	require.Equal(t, "Codertocat", ch.reviewContext.userLogin)
	require.Equal(t, "Hello-World", ch.reviewContext.repoName)
	require.Equal(t, "Codertocat", ch.reviewContext.repoOwner)
	require.Equal(t, "ec26c3e57ca3a959ca5aad62de7213c562f8c821", ch.reviewContext.headSHA)
}

func TestCheckInternal(t *testing.T) {
	env, err := environment.New(environment.Config{
		Client:           github.NewClient(nil),
		Token:            "12345",
		Reviewers:        `{"foo": ["bar", "baz"], "baz": ["foo", "car"], "bar": ["admin", "foo"]}`,
		DefaultReviewers: []string{"admin"},
		TeamSlug:         "team-name",
		Org:              "org",
	})
	require.NoError(t, err)

	tests := []struct {
		obj      map[string]review
		c        Check
		checkErr require.ErrorAssertionFunc
		desc     string
	}{
		{
			obj:      map[string]review{},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo"}, Environment: env, teamMembersFn: teamMembersTest, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with no reviews",
		},
		{
			obj: map[string]review{
				"bar": {name: "bar", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo"}, Environment: env, teamMembersFn: teamMembersTest, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with one one review and approval, but not all required approvals",
		},
		{
			obj: map[string]review{
				"bar": {name: "bar", status: "APPROVED", commitID: "1"},
				"baz": {name: "baz", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo"}, Environment: env, teamMembersFn: teamMembersTest, invalidate: invalidateTest},
			checkErr: require.NoError,
			desc:     "pull request with all required approvals",
		},
		{
			obj: map[string]review{
				"foo": {name: "foo", status: "APPROVED", commitID: "1"},
				"car": {name: "car", status: "COMMENTED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo"}, Environment: env, teamMembersFn: teamMembersTest, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with one approval and one comment review",
		},
		{
			obj: map[string]review{
				"admin": {name: "admin", status: "COMMENTED", commitID: "1"},
				"foo":   {name: "foo", status: "COMMENTED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "bar"}, Environment: env, teamMembersFn: teamMembersTest, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request does not have all required approvals",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.c.check(test.obj)
			test.checkErr(t, err)
		})
	}

}
func TestCheckExternal(t *testing.T) {
	tests := []struct {
		obj       map[string]review
		c         Check
		checkErr  require.ErrorAssertionFunc
		desc      string
		envConfig environment.Config
	}{
		{
			obj: map[string]review{
				"admin":  {name: "admin", status: "APPROVED", commitID: "1"},
				"admin2": {name: "admin2", status: "APPROVED", commitID: "2"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with all required approvals, commit hashes do not match",
			envConfig: environment.Config{
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin", "admin2"},
				Client:           github.NewClient(nil),
				Token:            "12345",
				Reviewers:        `{"ignored": ["bar", "baz"]}`,
			},
		},

		{
			obj: map[string]review{
				"admin":  {name: "admin", status: "APPROVED", commitID: "1"},
				"admin2": {name: "admin2", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
			checkErr: require.NoError,
			desc:     "pull request with all required approvals, commit hashes hashes match",
			envConfig: environment.Config{
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin", "admin2"},
				Client:           github.NewClient(nil),
				Token:            "12345",
				Reviewers:        `{"ignored": ["bar", "baz"]}`,
			},
		},
		{
			obj: map[string]review{
				"admin":  {name: "admin", status: "APPROVED", commitID: "1"},
				"admin2": {name: "admin2", status: "COMMENTED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with no required approvals, commit hashes match",
			envConfig: environment.Config{
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin", "admin2"},
				Client:           github.NewClient(nil),
				Token:            "12345",
				Reviewers:        `{"ignored": ["bar", "baz"]}`,
			},
		},
		{
			obj: map[string]review{
				"admin":  {name: "admin", status: "APPROVED", commitID: "1"},
				"admin2": {name: "admin2", status: "COMMENTED", commitID: "2"},
			},
			c:        Check{reviewContext: &ReviewContext{userLogin: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with some required approvals, commit hashes match",
			envConfig: environment.Config{
				TeamSlug:         "team-name",
				Org:              "org",
				DefaultReviewers: []string{"admin", "admin3"},
				Client:           github.NewClient(nil),
				Token:            "12345",
				Reviewers:        `{"ignored": ["bar", "baz"]}`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			newEnv, err := environment.New(test.envConfig)
			require.NoError(t, err)
			test.c.Environment = newEnv

			err = test.c.check(test.obj)
			test.checkErr(t, err)
		})
	}

}

func teamMembersTestExternal(org, slug string, cl *github.Client) ([]string, error) {
	return []string{}, nil
}

func teamMembersTest(org, slug string, cl *github.Client) ([]string, error) {
	return []string{"foo", "bar"}, nil
}

func invalidateTest(repoOwner, repoName string, number int, reviews map[string]review, clt *github.Client) error {
	return nil
}

const (
	invalidString = `{
		"action": "opened",
		"number": 2,
		"pull_request": {
		  "url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "id": 279147437,
		  "node_id": "MDExOlB1bGxSZXF1ZXN0Mjc5MTQ3NDM3",
		  "html_url": "https://github.com/Codertocat/Hello-World/pull/2",
		  "diff_url": "https://github.com/Codertocat/Hello-World/pull/2.diff",
		  "patch_url": "https://github.com/Codertocat/Hello-World/pull/2.patch",
		  "issue_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2",
		  "number": 2,
		  "state": "open",
		  "locked": false,
		  "title": "Update the README with new information.",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  },
		  "body": "This is a pretty simple change that we need to pull into master.",
		  "created_at": "2019-05-15T15:20:33Z",
		  "updated_at": "2019-05-15T15:20:33Z",
		  "closed_at": null,
		  "merged_at": null,
		  "merge_commit_sha": null,
		  "assignee": null,
		  "assignees": [],
		  "requested_reviewers": [],
		  "requested_teams": [],
		  "labels": [],
		  "milestone": null,
		  "commits_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/commits",
		  "review_comments_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/comments",
		  "review_comment_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/comments{/number}",
		  "comments_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2/comments",
		  "statuses_url": "https://api.github.com/repos/Codertocat/Hello-World/statuses/ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		  "author_association": "OWNER",
		  "draft": false,
		  "merged": false,
		  "mergeable": null,
		  "rebaseable": null,
		  "mergeable_state": "unknown",
		  "merged_by": null,
		  "comments": 0,
		  "review_comments": 0,
		  "maintainer_can_modify": false,
		  "commits": 1,
		  "additions": 1,
		  "deletions": 1,
		  "changed_files": 1
		},
		"repository": {
		  "id": 186853002,
		  "node_id": "MDEwOlJlcG9zaXRvcnkxODY4NTMwMDI=",
		  "name": "Hello-World",
		  "full_name": "Codertocat/Hello-World",
		  "private": false,
		  "owner": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  }
		}
	}`

	validString = `{
		"action": "submitted",
		"review": {
		  "id": 237895671,
		  "node_id": "MDE3OlB1bGxSZXF1ZXN0UmV2aWV3MjM3ODk1Njcx",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  },
		  "body": null,
		  "commit_id": "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		  "submitted_at": "2019-05-15T15:20:38Z",
		  "state": "commented",
		  "html_url": "https://github.com/Codertocat/Hello-World/pull/2#pullrequestreview-237895671",
		  "pull_request_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "author_association": "OWNER",
		  "_links": {
			"html": {
			  "href": "https://github.com/Codertocat/Hello-World/pull/2#pullrequestreview-237895671"
			},
			"pull_request": {
			  "href": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2"
			}
		  }
		},
		"pull_request": {
		  "url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "id": 279147437,
		  "node_id": "MDExOlB1bGxSZXF1ZXN0Mjc5MTQ3NDM3",
		  "html_url": "https://github.com/Codertocat/Hello-World/pull/2",
		  "diff_url": "https://github.com/Codertocat/Hello-World/pull/2.diff",
		  "patch_url": "https://github.com/Codertocat/Hello-World/pull/2.patch",
		  "issue_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2",
		  "number": 2,
		  "state": "open",
		  "locked": false,
		  "title": "Update the README with new information.",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  },
		  "body": "This is a pretty simple change that we need to pull into master.",
		  "created_at": "2019-05-15T15:20:33Z",
		  "updated_at": "2019-05-15T15:20:38Z",
		  "closed_at": null,
		  "merged_at": null,
		  "merge_commit_sha": "c4295bd74fb0f4fda03689c3df3f2803b658fd85",
		  "assignee": null,
		  "assignees": [],
		  "requested_reviewers": [],
		  "requested_teams": [],
		  "labels": [],
		  "milestone": null,
		  "commits_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/commits",
		  "review_comments_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/comments",
		  "review_comment_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/comments{/number}",
		  "comments_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2/comments",
		  "statuses_url": "https://api.github.com/repos/Codertocat/Hello-World/statuses/ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		  "head": {
			"label": "Codertocat:changes",
			"ref": "changes",
			"sha": "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
			"user": {
			  "login": "Codertocat",
			  "id": 21031067,
			  "node_id": "MDQ6VXNlcjIxMDMxMDY3",
			  "avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			  "gravatar_id": "",
			  "url": "https://api.github.com/users/Codertocat",
			  "html_url": "https://github.com/Codertocat",
			  "followers_url": "https://api.github.com/users/Codertocat/followers",
			  "following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			  "gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			  "starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			  "subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			  "organizations_url": "https://api.github.com/users/Codertocat/orgs",
			  "repos_url": "https://api.github.com/users/Codertocat/repos",
			  "events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			  "received_events_url": "https://api.github.com/users/Codertocat/received_events",
			  "type": "User",
			  "site_admin": false
			},
			"repo": {
			  "id": 186853002,
			  "node_id": "MDEwOlJlcG9zaXRvcnkxODY4NTMwMDI=",
			  "name": "Hello-World",
			  "full_name": "Codertocat/Hello-World",
			  "private": false,
			  "owner": {
				"login": "Codertocat",
				"id": 21031067,
				"node_id": "MDQ6VXNlcjIxMDMxMDY3",
				"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
				"gravatar_id": "",
				"url": "https://api.github.com/users/Codertocat",
				"html_url": "https://github.com/Codertocat",
				"followers_url": "https://api.github.com/users/Codertocat/followers",
				"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
				"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
				"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
				"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
				"organizations_url": "https://api.github.com/users/Codertocat/orgs",
				"repos_url": "https://api.github.com/users/Codertocat/repos",
				"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
				"received_events_url": "https://api.github.com/users/Codertocat/received_events",
				"type": "User",
				"site_admin": false
			  }
			}
		  },
		  "author_association": "OWNER"
		},
		"repository": {
		  "id": 186853002,
		  "node_id": "MDEwOlJlcG9zaXRvcnkxODY4NTMwMDI=",
		  "name": "Hello-World",
		  "full_name": "Codertocat/Hello-World",
		  "private": false,
		  "owner": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  }
		}
	}`

	invalidStringNoLogin = `{
		"action": "submitted",
		"review": {
		  "id": 237895671,
		  "node_id": "MDE3OlB1bGxSZXF1ZXN0UmV2aWV3MjM3ODk1Njcx",
		  "user": {
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  },
		  "body": null,
		  "commit_id": "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		  "submitted_at": "2019-05-15T15:20:38Z",
		  "state": "commented",
		  "html_url": "https://github.com/Codertocat/Hello-World/pull/2#pullrequestreview-237895671",
		  "pull_request_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "author_association": "OWNER",
		  "_links": {
			"html": {
			  "href": "https://github.com/Codertocat/Hello-World/pull/2#pullrequestreview-237895671"
			},
			"pull_request": {
			  "href": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2"
			}
		  }
		},
		"pull_request": {
		  "url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "id": 279147437,
		  "node_id": "MDExOlB1bGxSZXF1ZXN0Mjc5MTQ3NDM3",
		  "html_url": "https://github.com/Codertocat/Hello-World/pull/2",
		  "diff_url": "https://github.com/Codertocat/Hello-World/pull/2.diff",
		  "patch_url": "https://github.com/Codertocat/Hello-World/pull/2.patch",
		  "issue_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2",
		  "number": 2,
		  "state": "open",
		  "locked": false,
		  "title": "Update the README with new information.",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  },
		  "body": "This is a pretty simple change that we need to pull into master.",
		  "created_at": "2019-05-15T15:20:33Z",
		  "updated_at": "2019-05-15T15:20:38Z",
		  "closed_at": null,
		  "merged_at": null,
		  "merge_commit_sha": "c4295bd74fb0f4fda03689c3df3f2803b658fd85",
		  "assignee": null,
		  "assignees": [],
		  "requested_reviewers": [],
		  "requested_teams": [],
		  "labels": [],
		  "milestone": null,
		  "commits_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/commits",
		  "review_comments_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2/comments",
		  "review_comment_url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/comments{/number}",
		  "comments_url": "https://api.github.com/repos/Codertocat/Hello-World/issues/2/comments",
		  "statuses_url": "https://api.github.com/repos/Codertocat/Hello-World/statuses/ec26c3e57ca3a959ca5aad62de7213c562f8c821",
		  "head": {
			"label": "Codertocat:changes",
			"ref": "changes",
			"sha": "ec26c3e57ca3a959ca5aad62de7213c562f8c821",
			"user": {
			  "login": "Codertocat",
			  "id": 21031067,
			  "node_id": "MDQ6VXNlcjIxMDMxMDY3",
			  "avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			  "gravatar_id": "",
			  "url": "https://api.github.com/users/Codertocat",
			  "html_url": "https://github.com/Codertocat",
			  "followers_url": "https://api.github.com/users/Codertocat/followers",
			  "following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			  "gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			  "starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			  "subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			  "organizations_url": "https://api.github.com/users/Codertocat/orgs",
			  "repos_url": "https://api.github.com/users/Codertocat/repos",
			  "events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			  "received_events_url": "https://api.github.com/users/Codertocat/received_events",
			  "type": "User",
			  "site_admin": false
			},
			"repo": {
			  "id": 186853002,
			  "node_id": "MDEwOlJlcG9zaXRvcnkxODY4NTMwMDI=",
			  "name": "Hello-World",
			  "full_name": "Codertocat/Hello-World",
			  "private": false,
			  "owner": {
				"login": "Codertocat",
				"id": 21031067,
				"node_id": "MDQ6VXNlcjIxMDMxMDY3",
				"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
				"gravatar_id": "",
				"url": "https://api.github.com/users/Codertocat",
				"html_url": "https://github.com/Codertocat",
				"followers_url": "https://api.github.com/users/Codertocat/followers",
				"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
				"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
				"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
				"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
				"organizations_url": "https://api.github.com/users/Codertocat/orgs",
				"repos_url": "https://api.github.com/users/Codertocat/repos",
				"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
				"received_events_url": "https://api.github.com/users/Codertocat/received_events",
				"type": "User",
				"site_admin": false
			  }
			}
		  },
		  "author_association": "OWNER"
		},
		"repository": {
		  "id": 186853002,
		  "node_id": "MDEwOlJlcG9zaXRvcnkxODY4NTMwMDI=",
		  "name": "Hello-World",
		  "full_name": "Codertocat/Hello-World",
		  "private": false,
		  "owner": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			"gravatar_id": "",
			"url": "https://api.github.com/users/Codertocat",
			"html_url": "https://github.com/Codertocat",
			"followers_url": "https://api.github.com/users/Codertocat/followers",
			"following_url": "https://api.github.com/users/Codertocat/following{/other_user}",
			"gists_url": "https://api.github.com/users/Codertocat/gists{/gist_id}",
			"starred_url": "https://api.github.com/users/Codertocat/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/Codertocat/subscriptions",
			"organizations_url": "https://api.github.com/users/Codertocat/orgs",
			"repos_url": "https://api.github.com/users/Codertocat/repos",
			"events_url": "https://api.github.com/users/Codertocat/events{/privacy}",
			"received_events_url": "https://api.github.com/users/Codertocat/received_events",
			"type": "User",
			"site_admin": false
		  }
		}
	}`
)
