package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/nguyenvanduocit/executils"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

func main() {

	// prepare the arguments
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	shouldCommit := false
	flag.BoolVar(&shouldCommit, "commit", false, "commit the changes")

	flag.Parse()

	// prepare the diff
	diff, err := getDiff()
	if err != nil {
		fmt.Println(errors.WithMessage(err, "failed to get diff"))
		os.Exit(1)
	}

	if diff == "" {
		if isDirty() {
			fmt.Println("The repo is dirty but no thing was staged. Please stage your changes and try again")
		} else {
			fmt.Println("Nothing to commit")
		}
		os.Exit(0)
	}

	// prepare the client
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	client := gpt3.NewClient(apiKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

	commitPrompt := generateCommitPrompt(diff)

	commitMessage, err := complete(ctx, client, commitPrompt)
	if err != nil {
		fmt.Println(errors.WithMessage(err, "failed to generate commit message"))
		os.Exit(1)
	}

	if shouldCommit {
		if err := commit(commitMessage); err != nil {
			fmt.Println(errors.WithMessage(err, "failed to commit"))
			os.Exit(1)
		}

		fmt.Println("Committed with message:", commitMessage)
		return
	}

	fmt.Println(commitMessage)
}

func complete(ctx context.Context, client gpt3.Client, prompt string) (string, error) {
	resp, err := client.Completion(ctx, gpt3.CompletionRequest{
		Prompt: []string{
			prompt,
		},
		MaxTokens: gpt3.IntPtr(100),
	})

	if err != nil {
		return "", errors.WithMessage(err, "failed to complete")
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no choice was returned")
	}

	return strings.TrimSpace(resp.Choices[0].Text), nil
}

// commit commits the changes
func commit(message string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("commit", "-m", message),
	)

	return nil
}

// generateCommitPrompt generates the prompt for the commit message. This prompt use to instruct the AI that we want to generate a commit message that follows the conventional commit format
func generateCommitPrompt(diff string) string {
	return "Write a short conventional commit message for these changes:\n\n```\n" + diff + "\n```"
}

// getDiff returns the diff of the current branch
func getDiff() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	out := strings.Builder{}
	executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("diff", "--cached"),
		executils.WithStdOut(&out),
	)

	return strings.TrimSpace(out.String()), nil
}

// isDirty returns true if the repo is dirty
func isDirty() bool {
	workingDir, err := os.Getwd()
	if err != nil {
		return false
	}

	out := strings.Builder{}

	executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("diff"),
		executils.WithStdOut(&out),
	)

	return out.Len() > 0
}
