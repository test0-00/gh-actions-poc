package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/google/go-github/v37/github"
)

func main() {
	err := verifySig()
	if err != nil {
		log.Fatal(err)
	}
}

func verifySig() error {
	client := github.NewClient(nil)
	commit, _, err := client.Repositories.GetCommit(context.TODO(), "gravitational", "teleport", "f4ee52191cce728dd19ddd34c72bbe8858a281db") //api request
	if err != nil {
		return err
	}

	signature := *commit.Commit.Verification.Signature
	payloadData := *commit.Commit.Verification.Payload

	// creating file
	dataFile, err := ioutil.TempFile(".", "data")
	if err != nil {
		log.Fatal(err)
	}
	// Remember to clean up the file afterwards
	defer os.Remove(dataFile.Name())

	// Example writing to the file
	dataText := []byte(payloadData)
	if _, err = dataFile.Write(dataText); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}

	// Close the file
	if err := dataFile.Close(); err != nil {
		log.Fatal(err)
	}

	// creating file
	sigFile, err := ioutil.TempFile(".", "signature")
	if err != nil {
		log.Fatal(err)
	}
	// Remember to clean up the file afterwards
	//defer os.Remove(sigFile.Name())
	// Example writing to the file
	signatureText := []byte(signature)
	if _, err = sigFile.Write(signatureText); err != nil {
		log.Fatal("Failed to write to temporary file", err)
	}
	// Close the file
	if err := sigFile.Close(); err != nil {
		log.Fatal(err)
	}
	cmd := exec.Command("gpg", "--verify", "signature567710138", dataFile.Name())
	var errout bytes.Buffer
	cmd.Stderr = &errout

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			log.Fatal(exitError.ExitCode())
		}
	}
	fmt.Println(errout.String())

	return nil
}
