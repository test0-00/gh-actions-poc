package assign

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"

	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
)

func TestNewAssign(t *testing.T) {
	env, err := environment.New(environment.Config{
		Client:    github.NewClient(nil),
		Token:     "12345",
		Reviewers: `{"foo": ["bar", "baz"]}`,
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

	f, err := ioutil.TempFile("", "assign")
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
	f, err = ioutil.TempFile("", "assign")
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
	f, err = ioutil.TempFile("", "invalid-assign")
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

// TestNewPullRequestContextValid tests the unmarshalling of a valid pull request event
func TestPullRequestContextValid(t *testing.T) {
	ctx, err := newPullRequestContext([]byte(validString))
	require.NoError(t, err)
	require.Equal(t, 2, ctx.number)
	require.Equal(t, "Codertocat", ctx.userLogin)
	require.Equal(t, "Hello-World", ctx.repoName)
	require.Equal(t, "Codertocat", ctx.repoOwner)

}

// TestPullRequestContextInvalid tests the unmarshalling of an event that is not a pull request (i.e. review event)
func TestPullRequestContextInvalid(t *testing.T) {
	prCtx, err := newPullRequestContext([]byte(invalidString))
	require.Error(t, err)
	require.Nil(t, prCtx)

	prCtx, err = newPullRequestContext([]byte(""))
	require.Error(t, err)
	require.Nil(t, prCtx)
}

func TestAssign(t *testing.T) {
	env, err := environment.New(environment.Config{
		Client:    github.NewClient(nil),
		Token:     "12345",
		Reviewers: `{"foo": ["bar", "baz"], "baz": ["foo", "car"], "bar": ["admin", "foo"]}`,
	})
	require.NoError(t, err)

	tests := []struct {
		obj      map[string]bool
		env      Assign
		checkErr require.ErrorAssertionFunc
		desc     string
	}{
		{
			obj:      map[string]bool{},
			env:      Assign{pullContext: &PullRequestContext{userLogin: "foo"}, Environment: env},
			checkErr: require.Error,
			desc:     "no reviewers have been assigned",
		},
		{
			obj: map[string]bool{
				"bar": true,
				"baz": true,
			},
			env: Assign{pullContext: &PullRequestContext{userLogin: "foo"}, Environment: env},

			checkErr: require.NoError,
			desc:     "assigning was successful",
		},
		{
			obj: map[string]bool{
				"bar": true,
			},
			env:      Assign{pullContext: &PullRequestContext{userLogin: "random"}, Environment: env},
			checkErr: require.Error,
			desc:     "user does not exist in assigners",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.env.assign(test.obj)
			test.checkErr(t, err)
		})
	}
}

const (
	validString = `{
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

	invalidString = `{
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
)
