package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const (
	pullRequestNumber = "PULL_REQUEST_NUMBER"
	pullRequestAuthor = "PULL_REQUEST_AUTHOR"
	repoOwner         = "REPOSITORY_OWNER"
	repoName          = "REPOSITORY_NAME"
	githubToken       = "GITHUB_TOKEN"
	// ASSIGN is the argument to assign reviewers
	ASSIGN = "assign-reviewers"
	// CHECK is the argument to check reviewers
	CHECK = "check-reviewers"
)

func main() {
	// path := os.Getenv("GITHUB_EVENT_PATH")
	// jsonFile, err := os.Open(path)
	// if err != nil {
	// 	panic(err)
	// }
	// b, err := ioutil.ReadAll(jsonFile)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Print(string(b))
	var url string

	// reviewers := map[string][]string{
	// 	"quinqu":    []string{"0xblush", "russjones"},
	// 	"russjones": []string{"0xblush", "quinqu"},
	// }

	args := os.Args[1:]
	if len(args) != 1 {
		panic("exactly one argument needed \nassign-reviewers or check-reviewers")
	}

	switch args[0] {
	case ASSIGN:
		url = constructRequestReviewerEndpoint(os.Getenv(repoOwner), os.Getenv(repoName), os.Getenv(pullRequestNumber))
		fmt.Println("URL:>", url)

		var jsonStr = []byte(`{"reviewers":["0xblush", "russjones"]}`)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(githubToken)))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		fmt.Println("response Status:", resp.Status)
		fmt.Println("response Headers:", resp.Header)
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	case CHECK:
		// this case on review
		url = getReviewersEndpoint(os.Getenv(repoOwner), os.Getenv(repoName), os.Getenv(pullRequestNumber))

	default:
		panic("invalid input")
	}

	// ----- Get a repo -------
	// url := getRepository(os.Getenv(repoOwner), os.Getenv(repoName))
	// req, err := http.NewRequest("GET", url, nil)
	// req.Header.Set("Accept", "application/vnd.github.v3+json")
	// req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", os.Getenv(githubToken)))
	// req.Header.Set("Content-Type", "application/json")
	// ------------------------

}

func constructRequestReviewerEndpoint(owner, repoName, pullRequestNum string) string {
	name := strings.Split(repoName, "/")[1]
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/requested_reviewers", owner, name, pullRequestNum)
}

func getReviewersEndpoint(owner, repo, pullRequestNum string) string {
	name := strings.Split(repoName, "/")[1]
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%s/reviews", owner, name, pullRequestNum)

}

// change