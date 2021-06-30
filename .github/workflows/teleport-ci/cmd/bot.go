package main

import (
	"context"
	"log"
	"os"

	"../pkg/assign"


	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	GITHUBEVENTPATH = "GITHUB_EVENT_PATH"
	TOKEN = "GITHUB_TOKEN"
)

func main() {
	args := os.Args[1:]
	if len(args) != 1 {
		panic("one argument needed \nassign-reviewers or check-reviewers")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv(TOKEN)},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	path := os.Getenv(GITHUBEVENTPATH)
	token := os.Getenv(TOKEN)
	reviewers := os.Getenv("ASSIGNMENTS")

	cfg := assign.Config{
		Client:    client,
		EventPath: path,
		Token:     token,
		Reviewers: reviewers,
	}

	switch args[0] {
	case "assign-reviewers":
		log.Println("Assigning...")

		env, err := assign.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		err = env.Assign()
		if err != nil {
			log.Fatal(err)
		}

	case "check-reviewers":
		log.Println("Checking...")
		
	}

}
