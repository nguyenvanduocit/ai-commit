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

	temperature := flag.Float64("t", 0.7, "The temperature of the model. Higher temperature results in more random completions. Acceptable values are 0.0 to 1.0, inclusive.")

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

	client := gpt3.NewClient(apiKey, gpt3.WithDefaultEngine("text-chat-davinci-002-20221122"))

	commitMessage := ""
	promptAdjustment := "short, simple, clear"
	for {
		// prepare the client
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

		commitPrompt := generateCommitPrompt(diff, promptAdjustment)

		commitMessage, err = complete(ctx, client, float32(*temperature), commitPrompt)
		if err != nil {
			fmt.Println(errors.WithMessage(err, "failed to generate commit message"))
			os.Exit(1)
		}

		fmt.Printf("\n(temperature = %.1f) => %s\n\n", *temperature, commitMessage)

		fmt.Print("Does it fit?  \nor try new temperature (y/0-1): ")

		var input string
		fmt.Scanln(&input)

		if input == "1" {
			continue
		}

		if input == "y" {
			// ask for the type
			fmt.Print("Commit prefix (optional): ")
			fmt.Scanln(&input)

			if input != "" {
				commitMessage = input + ": " + commitMessage
			}

			break
		}

		if input == "" {
			continue
		}

		// check if the input is a float number
		if _, err := fmt.Sscanf(input, "%f", temperature); err != nil {
			fmt.Println("Invalid input")
			break
		}

		if *temperature < 0 || *temperature > 1 {
			fmt.Println("Temperature must be between 0 and 1")
			break
		}
	}

	if err := commit(commitMessage); err != nil {
		fmt.Println(errors.WithMessage(err, "failed to commit"))
		os.Exit(1)
	}

	fmt.Println("Commit successfully")
}

func complete(ctx context.Context, client gpt3.Client, temperature float32, prompt string) (string, error) {
	resp, err := client.Completion(ctx, gpt3.CompletionRequest{
		Prompt: []string{
			prompt,
		},
		Temperature: gpt3.Float32Ptr(temperature),
		MaxTokens:   gpt3.IntPtr(100),
	})

	if err != nil {
		return "", errors.WithMessage(err, "failed to complete")
	}

	if len(resp.Choices) == 0 {
		return "", errors.New("no choice was returned")
	}

	answer := strings.TrimSpace(resp.Choices[0].Text)
	answer = strings.Trim(answer, "\"")

	return answer, nil
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
func generateCommitPrompt(diff, promptAdjustment string) string {
	return "Write a " + promptAdjustment + " commit message for following diff output:" + ":\n\n```\n" + diff + "\n```"
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
		executils.WithArgs("diff", "--cached", "--unified=0"),
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
