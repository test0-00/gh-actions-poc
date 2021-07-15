package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/go-github/v37/github"
	ci "github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/assign"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/check"
	"github.com/gravitational/gh-actions-poc/.github/workflows/teleport-ci/pkg/environment"
	"golang.org/x/oauth2"
)

func main() {
	var token = flag.String("token", "", "token is the Github authentication token.")
	var assignments = flag.String("assignments", "", "assigners is a string representing a json object that maps authors to reviewers.")

	flag.Parse()

	if *token == "" {
		log.Fatal("missing authentication token.")
	}
	if *assignments == "" {
		log.Fatal("missing assignments string.")
	}

	subcommand := os.Args[len(os.Args)-1]

	// Creating and authenticating the Github client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Getting event object path and token
	path := os.Getenv(ci.GITHUBEVENTPATH)

	env, err := environment.New(environment.Config{Client: client,
		Token:     *token,
		Reviewers: *assignments, 
		TeamSlug: "dev-team",
		Org: "gravitational", 
		DefaultReviewers: []string{"quinqu"},
	})
	if err != nil {
		log.Fatal(err)
	}

	switch subcommand {
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
		log.Print("Assign completed.")

	case ci.CHECK:
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
		log.Print("Check completed.")
	default:
		log.Fatalf("Unknown subcommand: %v", subcommand)
	}

}
