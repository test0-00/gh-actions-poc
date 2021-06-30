package assign

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPullRequestValid(t *testing.T) {
	str := []byte(fmt.Sprint(`{
		"pull_request": {
		  "url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "id": 279147437,
		  "number": 2,
		  "state": "open",
		  "locked": false,
		  "title": "Update the README with new information.",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
			"type": "User",
			"site_admin": false
		  },
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
				"type": "User",
				"site_admin": false
			  }
			}
		  },
		  "base": {
			"label": "Codertocat:master",
			"ref": "master",
			"sha": "f95f852bd8fca8fcc58a9a2d6c842781e32a215e",
			"user": {
			  "login": "Codertocat",
			  "id": 21031067,
			  "node_id": "MDQ6VXNlcjIxMDMxMDY3",
			  "avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			  "gravatar_id": "",
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
				"type": "User",
				"site_admin": false
			  }
			}
		  },
		  "_links": {
		  },
		  "author_association": "OWNER"
		}
	  }`))
	env := &Environment{}
	err := env.newPullRequest([]byte(str))
	require.NoError(t, err)
}

func TestPullRequestInvalid(t *testing.T) {
	str := []byte(fmt.Sprint(`{
		"pull_request": {
		  "url": "https://api.github.com/repos/Codertocat/Hello-World/pulls/2",
		  "id": 279147437,
		  "state": "open",
		  "locked": false,
		  "title": "Update the README with new information.",
		  "user": {
			"login": "Codertocat",
			"id": 21031067,
			"node_id": "MDQ6VXNlcjIxMDMxMDY3",
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
			  "type": "User",
			  "site_admin": false
			},
		  },
		  "base": {
			"label": "Codertocat:master",
			"ref": "master",
			"sha": "f95f852bd8fca8fcc58a9a2d6c842781e32a215e",
			"user": {
			  "login": "Codertocat",
			  "id": 21031067,
			  "node_id": "MDQ6VXNlcjIxMDMxMDY3",
			  "avatar_url": "https://avatars1.githubusercontent.com/u/21031067?v=4",
			  "gravatar_id": "",
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
				"type": "User",
				"site_admin": false
			  }
			}
		  },
		  "_links": {
		  },
		  "author_association": "OWNER"
		}
	  }`))
	env := &Environment{}
	err := env.newPullRequest([]byte(str))
	require.Error(t, err)

}

func TestUnmarshalReviewers(t *testing.T) {
	tests := []struct {
		obj      string
		expected map[string][]string
		checkErr require.ErrorAssertionFunc
		desc     string
	}{
		{
			obj:      "",
			expected: nil,
			checkErr: require.Error,
			desc:     "empty object",
		},
		{
			obj: `{"foo":["bar"]}`,
			expected: map[string][]string{
				"foo": {"bar"},
			},
			checkErr: require.NoError,
			desc:     "valid user",
		},
		{
			obj:      `{"bar":"foo"}`,
			expected: nil,
			checkErr: require.Error,
			desc:     "invalid object format",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {

			res, err := unmarshalReviewers(test.obj)
			test.checkErr(t, err)
			require.EqualValues(t, test.expected, res)
		})
	}

}

func TestAssign(t *testing.T) {
	m := map[string][]string{
		"foo": {"bar", "baz"},
		"baz": {"foo", "car"},
		"bar": {"admin", "foo"},
	}

	tests := []struct {
		obj      map[string]User
		env      Environment
		checkErr require.ErrorAssertionFunc
		desc     string
	}{
		{
			obj:      map[string]User{},
			env:      Environment{PullRequest: PullRequest{User: User{Login: ""}}, Secrets: Secrets{Assigners: m}},
			checkErr: require.Error,
			desc:     "empty user and map",
		},
		{
			obj: map[string]User{
				"bar": {Login: "bar"},
				"baz": {Login: "baz"},
			},
			env:      Environment{PullRequest: PullRequest{User: User{Login: "foo"}}, Secrets: Secrets{Assigners: m}},
			checkErr: require.NoError,
			desc:     "assigning was successful",
		},
		{
			obj: map[string]User{
				"bar": {Login: "bar"},
			},
			env:      Environment{PullRequest: PullRequest{User: User{Login: "random"}}, Secrets: Secrets{Assigners: m}},
			checkErr: require.Error,
			desc:     "user does not exist in map",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := test.env.assign(test.obj)
			test.checkErr(t, err)
		})
	}
}

type pullRequest struct {
	number             int
	requestedReviewers []string
}
