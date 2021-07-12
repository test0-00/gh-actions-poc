package main

import (
	"context"
	"log"
	"os"

	"github.com/google/go-github/github"
	ci "github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/assign"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/check"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"

	"golang.org/x/oauth2"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		panic("one argument needed \nassign-reviewers or check-reviewers")
	}
	// Creating and authenticating the Github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv(ci.TOKEN)},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	
	// Getting event object path and token
	path := os.Getenv(ci.GITHUBEVENTPATH)
	token := os.Getenv(ci.TOKEN)
	reviewers := os.Getenv(ci.ASSIGNMENTS)

	env, err := environment.New(environment.Config{Client: client,
		Token:     token,
		Reviewers: reviewers})
	if err != nil {
		log.Fatal(err)
	}

	switch args[0] {
	case ci.ASSIGN:
		log.Println("Assigning reviewers...")
		cfg := assign.Config{
			Environment: env,
			EventPath:   path,
		}
		assigner, err := assign.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		err = assigner.Assign()
		if err != nil {
			log.Fatal(err)
		}
		
	case "check-reviewers":
		log.Println("Checking reviewers...")
		cfg := check.Config{
			Environment: env,
			EventPath:   path,
		}
		checker, err := check.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		err = checker.Check()
		if err != nil {
			log.Fatal(err)
		}

	}

}
