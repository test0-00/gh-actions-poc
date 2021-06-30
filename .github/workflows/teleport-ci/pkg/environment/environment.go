package environment

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/google/go-github/github"
	"github.com/gravitational/trace"
)

// Environment ...
type Environment struct {
	PullRequest PullRequest `json:"pull_request"`
	Secrets     Secrets
	Client      *github.Client
}

// PullRequest ...
type PullRequest struct {
	Number int `json:"number"`
	User   User
	Head   Head `json:"head"`
}

// User ...
type User struct {
	Login string `json:"login"`
}

// Head ...
type Head struct {
	Repo Repo `json:"repo"`
}

// Repo ...
type Repo struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

// Owner ...
type Owner struct {
	Name string `json:"login"`
}

// Secrets ...
type Secrets struct {
	Assigners map[string][]string
	Token     string
}

type review struct {
	reviewer string
	status   string
}

// NewPullRequest unmarshals pull request metadata from json file given the path
func (e *Environment) NewPullRequest(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return trace.Wrap(err)
	}
	body, err := ioutil.ReadAll(file)
	if err != nil {
		return trace.Wrap(err)
	}
	return e.newPullRequest(body)
}

func (e *Environment) newPullRequest(body []byte) error {
	err := json.Unmarshal(body, e)
	if err != nil {
		return trace.Wrap(err)
	}
	pullReq := e.PullRequest
	if pullReq.Number == 0 || pullReq.User.Login == "" || pullReq.Head.Repo.Name == "" || pullReq.Head.Repo.Owner.Name == "" {
		return trace.BadParameter("insufficient data obatined")
	}
	return nil
}

func UnmarshalReviewers(str string) (map[string][]string, error) {
	if str == "" {
		return nil, trace.BadParameter("reviewers not found")
	}
	m := make(map[string][]string)

	err := json.Unmarshal([]byte(str), &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}