package main

import (
	"context"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/nguyenvanduocit/executils"
	"github.com/pkg/errors"
	"os"
	"strings"
)

func main() {

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	ctx := context.Background()
	client := gpt3.NewClient(apiKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

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

	resp, err := client.Completion(ctx, gpt3.CompletionRequest{
		Prompt: []string{
			generateCommitPrompt(diff),
		},
		MaxTokens: gpt3.IntPtr(100),
	})
	if err != nil {
		fmt.Println(errors.WithMessage(err, "failed to generate commit message"))
		os.Exit(1)
	}

	commitMessage := strings.TrimSpace(resp.Choices[0].Text)

	fmt.Print(commitMessage)
}

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
