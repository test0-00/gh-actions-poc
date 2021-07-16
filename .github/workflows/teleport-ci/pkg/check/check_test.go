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
	require.Equal(t, "Codertocat", ch.reviewContext.author)
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
	ch := Check{Environment: &environment.Environment{Client: github.NewClient(nil)}, action: "synchronize"}
	err := ch.setReviewContext([]byte(validStringSynchronize))
	require.NoError(t, err)
	require.Equal(t, 28, ch.reviewContext.number)
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
			c:        Check{reviewContext: &ReviewContext{author: "foo"}, Environment: env, teamMembersFn: teamMembersTestInternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with no reviews",
		},
		{
			obj: map[string]review{
				"bar": {name: "bar", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{author: "foo"}, Environment: env, teamMembersFn: teamMembersTestInternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with one one review and approval, but not all required approvals",
		},
		{
			obj: map[string]review{
				"bar": {name: "bar", status: "APPROVED", commitID: "1"},
				"baz": {name: "baz", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{author: "foo"}, Environment: env, teamMembersFn: teamMembersTestInternal, invalidate: invalidateTest},
			checkErr: require.NoError,
			desc:     "pull request with all required approvals",
		},
		{
			obj: map[string]review{
				"foo": {name: "foo", status: "APPROVED", commitID: "1"},
				"car": {name: "car", status: "COMMENTED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{author: "foo"}, Environment: env, teamMembersFn: teamMembersTestInternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with one approval and one comment review",
		},
		{
			obj: map[string]review{
				"admin": {name: "admin", status: "COMMENTED", commitID: "1"},
				"foo":   {name: "foo", status: "COMMENTED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{author: "bar"}, Environment: env, teamMembersFn: teamMembersTestInternal, invalidate: invalidateTest},
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
				"admin2": {name: "admin2", status: "APPROVED", commitID: "1"},
			},
			c:        Check{reviewContext: &ReviewContext{author: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
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
			c:        Check{reviewContext: &ReviewContext{author: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
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
			c:        Check{reviewContext: &ReviewContext{author: "foo", headSHA: "1"}, teamMembersFn: teamMembersTestExternal, invalidate: invalidateTest},
			checkErr: require.Error,
			desc:     "pull request with some required approvals, commit hashes do not match",
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

func teamMembersTestInternal(org, slug string, cl *github.Client) ([]string, error) {
	return []string{"foo", "bar"}, nil
}

func invalidateTest(repoOwner, repoName, msg string, number int, reviews map[string]review, clt *github.Client) error {
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

	validStringSynchronize = `{
		"action": "synchronize",
		"after": "ecabd9d97b218368ea47d17cd23815590b76e196",
		"before": "cbb23161d4c33d70189430d07957d2d66d42fc30",
		"number": 28,
		"organization": {
		  "avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
		  "description": "Unify access for SSH servers, Kubernetes clusters, web applications, and databases.",
		  "events_url": "https://api.github.com/orgs/gravitational/events",
		  "hooks_url": "https://api.github.com/orgs/gravitational/hooks",
		  "id": 10781132,
		  "issues_url": "https://api.github.com/orgs/gravitational/issues",
		  "login": "gravitational",
		  "members_url": "https://api.github.com/orgs/gravitational/members{/member}",
		  "node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
		  "public_members_url": "https://api.github.com/orgs/gravitational/public_members{/member}",
		  "repos_url": "https://api.github.com/orgs/gravitational/repos",
		  "url": "https://api.github.com/orgs/gravitational"
		},
		"pull_request": {
		  "_links": {
			"comments": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/28/comments"
			},
			"commits": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28/commits"
			},
			"html": {
			  "href": "https://github.com/gravitational/gh-actions-poc/pull/28"
			},
			"issue": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/28"
			},
			"review_comment": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/comments{/number}"
			},
			"review_comments": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28/comments"
			},
			"self": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28"
			},
			"statuses": {
			  "href": "https://api.github.com/repos/gravitational/gh-actions-poc/statuses/ecabd9d97b218368ea47d17cd23815590b76e196"
			}
		  },
		  "active_lock_reason": null,
		  "additions": 314565,
		  "assignee": null,
		  "assignees": [],
		  "author_association": "COLLABORATOR",
		  "auto_merge": null,
		  "base": {
			"label": "gravitational:master",
			"ref": "master",
			"repo": {
			  "allow_merge_commit": true,
			  "allow_rebase_merge": true,
			  "allow_squash_merge": true,
			  "archive_url": "https://api.github.com/repos/gravitational/gh-actions-poc/{archive_format}{/ref}",
			  "archived": false,
			  "assignees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/assignees{/user}",
			  "blobs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/blobs{/sha}",
			  "branches_url": "https://api.github.com/repos/gravitational/gh-actions-poc/branches{/branch}",
			  "clone_url": "https://github.com/gravitational/gh-actions-poc.git",
			  "collaborators_url": "https://api.github.com/repos/gravitational/gh-actions-poc/collaborators{/collaborator}",
			  "comments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/comments{/number}",
			  "commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/commits{/sha}",
			  "compare_url": "https://api.github.com/repos/gravitational/gh-actions-poc/compare/{base}...{head}",
			  "contents_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contents/{+path}",
			  "contributors_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contributors",
			  "created_at": "2021-05-06T16:56:44Z",
			  "default_branch": "master",
			  "delete_branch_on_merge": false,
			  "deployments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/deployments",
			  "description": null,
			  "disabled": false,
			  "downloads_url": "https://api.github.com/repos/gravitational/gh-actions-poc/downloads",
			  "events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/events",
			  "fork": false,
			  "forks": 1,
			  "forks_count": 1,
			  "forks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/forks",
			  "full_name": "gravitational/gh-actions-poc",
			  "git_commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/commits{/sha}",
			  "git_refs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/refs{/sha}",
			  "git_tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/tags{/sha}",
			  "git_url": "git://github.com/gravitational/gh-actions-poc.git",
			  "has_downloads": true,
			  "has_issues": true,
			  "has_pages": false,
			  "has_projects": true,
			  "has_wiki": true,
			  "homepage": null,
			  "hooks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/hooks",
			  "html_url": "https://github.com/gravitational/gh-actions-poc",
			  "id": 364979824,
			  "issue_comment_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/comments{/number}",
			  "issue_events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/events{/number}",
			  "issues_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues{/number}",
			  "keys_url": "https://api.github.com/repos/gravitational/gh-actions-poc/keys{/key_id}",
			  "labels_url": "https://api.github.com/repos/gravitational/gh-actions-poc/labels{/name}",
			  "language": "Go",
			  "languages_url": "https://api.github.com/repos/gravitational/gh-actions-poc/languages",
			  "license": null,
			  "merges_url": "https://api.github.com/repos/gravitational/gh-actions-poc/merges",
			  "milestones_url": "https://api.github.com/repos/gravitational/gh-actions-poc/milestones{/number}",
			  "mirror_url": null,
			  "name": "gh-actions-poc",
			  "node_id": "MDEwOlJlcG9zaXRvcnkzNjQ5Nzk4MjQ=",
			  "notifications_url": "https://api.github.com/repos/gravitational/gh-actions-poc/notifications{?since,all,participating}",
			  "open_issues": 9,
			  "open_issues_count": 9,
			  "owner": {
				"avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
				"events_url": "https://api.github.com/users/gravitational/events{/privacy}",
				"followers_url": "https://api.github.com/users/gravitational/followers",
				"following_url": "https://api.github.com/users/gravitational/following{/other_user}",
				"gists_url": "https://api.github.com/users/gravitational/gists{/gist_id}",
				"gravatar_id": "",
				"html_url": "https://github.com/gravitational",
				"id": 10781132,
				"login": "gravitational",
				"node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
				"organizations_url": "https://api.github.com/users/gravitational/orgs",
				"received_events_url": "https://api.github.com/users/gravitational/received_events",
				"repos_url": "https://api.github.com/users/gravitational/repos",
				"site_admin": false,
				"starred_url": "https://api.github.com/users/gravitational/starred{/owner}{/repo}",
				"subscriptions_url": "https://api.github.com/users/gravitational/subscriptions",
				"type": "Organization",
				"url": "https://api.github.com/users/gravitational"
			  },
			  "private": false,
			  "pulls_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls{/number}",
			  "pushed_at": "2021-07-15T18:35:54Z",
			  "releases_url": "https://api.github.com/repos/gravitational/gh-actions-poc/releases{/id}",
			  "size": 3466,
			  "ssh_url": "git@github.com:gravitational/gh-actions-poc.git",
			  "stargazers_count": 1,
			  "stargazers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/stargazers",
			  "statuses_url": "https://api.github.com/repos/gravitational/gh-actions-poc/statuses/{sha}",
			  "subscribers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscribers",
			  "subscription_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscription",
			  "svn_url": "https://github.com/gravitational/gh-actions-poc",
			  "tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/tags",
			  "teams_url": "https://api.github.com/repos/gravitational/gh-actions-poc/teams",
			  "trees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/trees{/sha}",
			  "updated_at": "2021-07-14T08:52:30Z",
			  "url": "https://api.github.com/repos/gravitational/gh-actions-poc",
			  "watchers": 1,
			  "watchers_count": 1
			},
			"sha": "385a4f19e99a35adeef42e8188036e3742ca0387",
			"user": {
			  "avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
			  "events_url": "https://api.github.com/users/gravitational/events{/privacy}",
			  "followers_url": "https://api.github.com/users/gravitational/followers",
			  "following_url": "https://api.github.com/users/gravitational/following{/other_user}",
			  "gists_url": "https://api.github.com/users/gravitational/gists{/gist_id}",
			  "gravatar_id": "",
			  "html_url": "https://github.com/gravitational",
			  "id": 10781132,
			  "login": "gravitational",
			  "node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
			  "organizations_url": "https://api.github.com/users/gravitational/orgs",
			  "received_events_url": "https://api.github.com/users/gravitational/received_events",
			  "repos_url": "https://api.github.com/users/gravitational/repos",
			  "site_admin": false,
			  "starred_url": "https://api.github.com/users/gravitational/starred{/owner}{/repo}",
			  "subscriptions_url": "https://api.github.com/users/gravitational/subscriptions",
			  "type": "Organization",
			  "url": "https://api.github.com/users/gravitational"
			}
		  },
		  "body": "",
		  "changed_files": 784,
		  "closed_at": null,
		  "comments": 0,
		  "comments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/28/comments",
		  "commits": 32,
		  "commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28/commits",
		  "created_at": "2021-07-15T18:06:04Z",
		  "deletions": 241,
		  "diff_url": "https://github.com/gravitational/gh-actions-poc/pull/28.diff",
		  "draft": false,
		  "head": {
			"label": "gravitational:jane/ci",
			"ref": "jane/ci",
			"repo": {
			  "allow_merge_commit": true,
			  "allow_rebase_merge": true,
			  "allow_squash_merge": true,
			  "archive_url": "https://api.github.com/repos/gravitational/gh-actions-poc/{archive_format}{/ref}",
			  "archived": false,
			  "assignees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/assignees{/user}",
			  "blobs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/blobs{/sha}",
			  "branches_url": "https://api.github.com/repos/gravitational/gh-actions-poc/branches{/branch}",
			  "clone_url": "https://github.com/gravitational/gh-actions-poc.git",
			  "collaborators_url": "https://api.github.com/repos/gravitational/gh-actions-poc/collaborators{/collaborator}",
			  "comments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/comments{/number}",
			  "commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/commits{/sha}",
			  "compare_url": "https://api.github.com/repos/gravitational/gh-actions-poc/compare/{base}...{head}",
			  "contents_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contents/{+path}",
			  "contributors_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contributors",
			  "created_at": "2021-05-06T16:56:44Z",
			  "default_branch": "master",
			  "delete_branch_on_merge": false,
			  "deployments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/deployments",
			  "description": null,
			  "disabled": false,
			  "downloads_url": "https://api.github.com/repos/gravitational/gh-actions-poc/downloads",
			  "events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/events",
			  "fork": false,
			  "forks": 1,
			  "forks_count": 1,
			  "forks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/forks",
			  "full_name": "gravitational/gh-actions-poc",
			  "git_commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/commits{/sha}",
			  "git_refs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/refs{/sha}",
			  "git_tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/tags{/sha}",
			  "git_url": "git://github.com/gravitational/gh-actions-poc.git",
			  "has_downloads": true,
			  "has_issues": true,
			  "has_pages": false,
			  "has_projects": true,
			  "has_wiki": true,
			  "homepage": null,
			  "hooks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/hooks",
			  "html_url": "https://github.com/gravitational/gh-actions-poc",
			  "id": 364979824,
			  "issue_comment_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/comments{/number}",
			  "issue_events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/events{/number}",
			  "issues_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues{/number}",
			  "keys_url": "https://api.github.com/repos/gravitational/gh-actions-poc/keys{/key_id}",
			  "labels_url": "https://api.github.com/repos/gravitational/gh-actions-poc/labels{/name}",
			  "language": "Go",
			  "languages_url": "https://api.github.com/repos/gravitational/gh-actions-poc/languages",
			  "license": null,
			  "merges_url": "https://api.github.com/repos/gravitational/gh-actions-poc/merges",
			  "milestones_url": "https://api.github.com/repos/gravitational/gh-actions-poc/milestones{/number}",
			  "mirror_url": null,
			  "name": "gh-actions-poc",
			  "node_id": "MDEwOlJlcG9zaXRvcnkzNjQ5Nzk4MjQ=",
			  "notifications_url": "https://api.github.com/repos/gravitational/gh-actions-poc/notifications{?since,all,participating}",
			  "open_issues": 9,
			  "open_issues_count": 9,
			  "owner": {
				"avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
				"events_url": "https://api.github.com/users/gravitational/events{/privacy}",
				"followers_url": "https://api.github.com/users/gravitational/followers",
				"following_url": "https://api.github.com/users/gravitational/following{/other_user}",
				"gists_url": "https://api.github.com/users/gravitational/gists{/gist_id}",
				"gravatar_id": "",
				"html_url": "https://github.com/gravitational",
				"id": 10781132,
				"login": "gravitational",
				"node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
				"organizations_url": "https://api.github.com/users/gravitational/orgs",
				"received_events_url": "https://api.github.com/users/gravitational/received_events",
				"repos_url": "https://api.github.com/users/gravitational/repos",
				"site_admin": false,
				"starred_url": "https://api.github.com/users/gravitational/starred{/owner}{/repo}",
				"subscriptions_url": "https://api.github.com/users/gravitational/subscriptions",
				"type": "Organization",
				"url": "https://api.github.com/users/gravitational"
			  },
			  "private": false,
			  "pulls_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls{/number}",
			  "pushed_at": "2021-07-15T18:35:54Z",
			  "releases_url": "https://api.github.com/repos/gravitational/gh-actions-poc/releases{/id}",
			  "size": 3466,
			  "ssh_url": "git@github.com:gravitational/gh-actions-poc.git",
			  "stargazers_count": 1,
			  "stargazers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/stargazers",
			  "statuses_url": "https://api.github.com/repos/gravitational/gh-actions-poc/statuses/{sha}",
			  "subscribers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscribers",
			  "subscription_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscription",
			  "svn_url": "https://github.com/gravitational/gh-actions-poc",
			  "tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/tags",
			  "teams_url": "https://api.github.com/repos/gravitational/gh-actions-poc/teams",
			  "trees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/trees{/sha}",
			  "updated_at": "2021-07-14T08:52:30Z",
			  "url": "https://api.github.com/repos/gravitational/gh-actions-poc",
			  "watchers": 1,
			  "watchers_count": 1
			},
			"sha": "ecabd9d97b218368ea47d17cd23815590b76e196",
			"user": {
			  "avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
			  "events_url": "https://api.github.com/users/gravitational/events{/privacy}",
			  "followers_url": "https://api.github.com/users/gravitational/followers",
			  "following_url": "https://api.github.com/users/gravitational/following{/other_user}",
			  "gists_url": "https://api.github.com/users/gravitational/gists{/gist_id}",
			  "gravatar_id": "",
			  "html_url": "https://github.com/gravitational",
			  "id": 10781132,
			  "login": "gravitational",
			  "node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
			  "organizations_url": "https://api.github.com/users/gravitational/orgs",
			  "received_events_url": "https://api.github.com/users/gravitational/received_events",
			  "repos_url": "https://api.github.com/users/gravitational/repos",
			  "site_admin": false,
			  "starred_url": "https://api.github.com/users/gravitational/starred{/owner}{/repo}",
			  "subscriptions_url": "https://api.github.com/users/gravitational/subscriptions",
			  "type": "Organization",
			  "url": "https://api.github.com/users/gravitational"
			}
		  },
		  "html_url": "https://github.com/gravitational/gh-actions-poc/pull/28",
		  "id": 690933440,
		  "issue_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/28",
		  "labels": [],
		  "locked": false,
		  "maintainer_can_modify": false,
		  "merge_commit_sha": "8c8dcf9cf0ead8f4f77212f8b991675e122a6f29",
		  "mergeable": null,
		  "mergeable_state": "unknown",
		  "merged": false,
		  "merged_at": null,
		  "merged_by": null,
		  "milestone": null,
		  "node_id": "MDExOlB1bGxSZXF1ZXN0NjkwOTMzNDQw",
		  "number": 28,
		  "patch_url": "https://github.com/gravitational/gh-actions-poc/pull/28.patch",
		  "rebaseable": null,
		  "requested_reviewers": [],
		  "requested_teams": [],
		  "review_comment_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/comments{/number}",
		  "review_comments": 0,
		  "review_comments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28/comments",
		  "state": "open",
		  "statuses_url": "https://api.github.com/repos/gravitational/gh-actions-poc/statuses/ecabd9d97b218368ea47d17cd23815590b76e196",
		  "title": "Jane/ci",
		  "updated_at": "2021-07-15T18:35:56Z",
		  "url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls/28",
		  "user": {
			"avatar_url": "https://avatars.githubusercontent.com/u/42625018?v=4",
			"events_url": "https://api.github.com/users/quinqu/events{/privacy}",
			"followers_url": "https://api.github.com/users/quinqu/followers",
			"following_url": "https://api.github.com/users/quinqu/following{/other_user}",
			"gists_url": "https://api.github.com/users/quinqu/gists{/gist_id}",
			"gravatar_id": "",
			"html_url": "https://github.com/quinqu",
			"id": 42625018,
			"login": "quinqu",
			"node_id": "MDQ6VXNlcjQyNjI1MDE4",
			"organizations_url": "https://api.github.com/users/quinqu/orgs",
			"received_events_url": "https://api.github.com/users/quinqu/received_events",
			"repos_url": "https://api.github.com/users/quinqu/repos",
			"site_admin": false,
			"starred_url": "https://api.github.com/users/quinqu/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/quinqu/subscriptions",
			"type": "User",
			"url": "https://api.github.com/users/quinqu"
		  }
		},
		"repository": {
		  "archive_url": "https://api.github.com/repos/gravitational/gh-actions-poc/{archive_format}{/ref}",
		  "archived": false,
		  "assignees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/assignees{/user}",
		  "blobs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/blobs{/sha}",
		  "branches_url": "https://api.github.com/repos/gravitational/gh-actions-poc/branches{/branch}",
		  "clone_url": "https://github.com/gravitational/gh-actions-poc.git",
		  "collaborators_url": "https://api.github.com/repos/gravitational/gh-actions-poc/collaborators{/collaborator}",
		  "comments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/comments{/number}",
		  "commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/commits{/sha}",
		  "compare_url": "https://api.github.com/repos/gravitational/gh-actions-poc/compare/{base}...{head}",
		  "contents_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contents/{+path}",
		  "contributors_url": "https://api.github.com/repos/gravitational/gh-actions-poc/contributors",
		  "created_at": "2021-05-06T16:56:44Z",
		  "default_branch": "master",
		  "deployments_url": "https://api.github.com/repos/gravitational/gh-actions-poc/deployments",
		  "description": null,
		  "disabled": false,
		  "downloads_url": "https://api.github.com/repos/gravitational/gh-actions-poc/downloads",
		  "events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/events",
		  "fork": false,
		  "forks": 1,
		  "forks_count": 1,
		  "forks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/forks",
		  "full_name": "gravitational/gh-actions-poc",
		  "git_commits_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/commits{/sha}",
		  "git_refs_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/refs{/sha}",
		  "git_tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/tags{/sha}",
		  "git_url": "git://github.com/gravitational/gh-actions-poc.git",
		  "has_downloads": true,
		  "has_issues": true,
		  "has_pages": false,
		  "has_projects": true,
		  "has_wiki": true,
		  "homepage": null,
		  "hooks_url": "https://api.github.com/repos/gravitational/gh-actions-poc/hooks",
		  "html_url": "https://github.com/gravitational/gh-actions-poc",
		  "id": 364979824,
		  "issue_comment_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/comments{/number}",
		  "issue_events_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues/events{/number}",
		  "issues_url": "https://api.github.com/repos/gravitational/gh-actions-poc/issues{/number}",
		  "keys_url": "https://api.github.com/repos/gravitational/gh-actions-poc/keys{/key_id}",
		  "labels_url": "https://api.github.com/repos/gravitational/gh-actions-poc/labels{/name}",
		  "language": "Go",
		  "languages_url": "https://api.github.com/repos/gravitational/gh-actions-poc/languages",
		  "license": null,
		  "merges_url": "https://api.github.com/repos/gravitational/gh-actions-poc/merges",
		  "milestones_url": "https://api.github.com/repos/gravitational/gh-actions-poc/milestones{/number}",
		  "mirror_url": null,
		  "name": "gh-actions-poc",
		  "node_id": "MDEwOlJlcG9zaXRvcnkzNjQ5Nzk4MjQ=",
		  "notifications_url": "https://api.github.com/repos/gravitational/gh-actions-poc/notifications{?since,all,participating}",
		  "open_issues": 9,
		  "open_issues_count": 9,
		  "owner": {
			"avatar_url": "https://avatars.githubusercontent.com/u/10781132?v=4",
			"events_url": "https://api.github.com/users/gravitational/events{/privacy}",
			"followers_url": "https://api.github.com/users/gravitational/followers",
			"following_url": "https://api.github.com/users/gravitational/following{/other_user}",
			"gists_url": "https://api.github.com/users/gravitational/gists{/gist_id}",
			"gravatar_id": "",
			"html_url": "https://github.com/gravitational",
			"id": 10781132,
			"login": "gravitational",
			"node_id": "MDEyOk9yZ2FuaXphdGlvbjEwNzgxMTMy",
			"organizations_url": "https://api.github.com/users/gravitational/orgs",
			"received_events_url": "https://api.github.com/users/gravitational/received_events",
			"repos_url": "https://api.github.com/users/gravitational/repos",
			"site_admin": false,
			"starred_url": "https://api.github.com/users/gravitational/starred{/owner}{/repo}",
			"subscriptions_url": "https://api.github.com/users/gravitational/subscriptions",
			"type": "Organization",
			"url": "https://api.github.com/users/gravitational"
		  },
		  "private": false,
		  "pulls_url": "https://api.github.com/repos/gravitational/gh-actions-poc/pulls{/number}",
		  "pushed_at": "2021-07-15T18:35:54Z",
		  "releases_url": "https://api.github.com/repos/gravitational/gh-actions-poc/releases{/id}",
		  "size": 3466,
		  "ssh_url": "git@github.com:gravitational/gh-actions-poc.git",
		  "stargazers_count": 1,
		  "stargazers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/stargazers",
		  "statuses_url": "https://api.github.com/repos/gravitational/gh-actions-poc/statuses/{sha}",
		  "subscribers_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscribers",
		  "subscription_url": "https://api.github.com/repos/gravitational/gh-actions-poc/subscription",
		  "svn_url": "https://github.com/gravitational/gh-actions-poc",
		  "tags_url": "https://api.github.com/repos/gravitational/gh-actions-poc/tags",
		  "teams_url": "https://api.github.com/repos/gravitational/gh-actions-poc/teams",
		  "trees_url": "https://api.github.com/repos/gravitational/gh-actions-poc/git/trees{/sha}",
		  "updated_at": "2021-07-14T08:52:30Z",
		  "url": "https://api.github.com/repos/gravitational/gh-actions-poc",
		  "watchers": 1,
		  "watchers_count": 1
		},
		"sender": {
		  "avatar_url": "https://avatars.githubusercontent.com/u/42625018?v=4",
		  "events_url": "https://api.github.com/users/quinqu/events{/privacy}",
		  "followers_url": "https://api.github.com/users/quinqu/followers",
		  "following_url": "https://api.github.com/users/quinqu/following{/other_user}",
		  "gists_url": "https://api.github.com/users/quinqu/gists{/gist_id}",
		  "gravatar_id": "",
		  "html_url": "https://github.com/quinqu",
		  "id": 42625018,
		  "login": "quinqu",
		  "node_id": "MDQ6VXNlcjQyNjI1MDE4",
		  "organizations_url": "https://api.github.com/users/quinqu/orgs",
		  "received_events_url": "https://api.github.com/users/quinqu/received_events",
		  "repos_url": "https://api.github.com/users/quinqu/repos",
		  "site_admin": false,
		  "starred_url": "https://api.github.com/users/quinqu/starred{/owner}{/repo}",
		  "subscriptions_url": "https://api.github.com/users/quinqu/subscriptions",
		  "type": "User",
		  "url": "https://api.github.com/users/quinqu"
		}
	}`
)
