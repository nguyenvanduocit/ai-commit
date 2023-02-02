package main

import (
	"context"
	"fmt"
	"github.com/PullRequestInc/go-gpt3"
	"github.com/nguyenvanduocit/executils"
	"github.com/pkg/errors"
	"log"
	"os"
	"strings"
)

func main() {

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatalln("Missing OPENAI_API_KEY")
	}

	ctx := context.Background()
	client := gpt3.NewClient(apiKey, gpt3.WithDefaultEngine(gpt3.TextDavinci003Engine))

	diff, err := getDiff()
	if err != nil {
		log.Fatalln(errors.WithMessage(err, "failed to get diff"))
	}

	resp, err := client.Completion(ctx, gpt3.CompletionRequest{
		Prompt: []string{
			generateCommitPrompt(diff),
		},
		MaxTokens: gpt3.IntPtr(100),
		Echo:      false,
	})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println(resp.Choices[0].Text)
}

func generateCommitPrompt(diff string) string {
	return "Write a short conventional commit message for this changes: \n\n ```\n" + diff + "\n```\n\n"
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
