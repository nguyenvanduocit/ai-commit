package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/nguyenvanduocit/executils"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

var messages = []*Message{
	{
		Role:    "user",
		Content: `You are a senior developer, you are using conventional commit to write commit message for this diff`,
	},
}

func main() {

	// prepare the arguments
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

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

	client := NewGptClient(apiKey)

	commitMessage := ""
	for {
		// prepare the client
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		messages = append(messages, &Message{
			Role:    "user",
			Content: diff,
		})

		commitMessage, err = client.ChatComplete(ctx, messages)
		if err != nil {
			fmt.Println(errors.WithMessage(err, "failed to generate commit message"))
			os.Exit(1)
		}

		fmt.Printf("\nai-commiter: %s\n\n", commitMessage)

		fmt.Print("User: ")

		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(errors.WithMessage(err, "failed to read user input"))
			os.Exit(1)
		}

		if line == "" {
			break
		}

		isAgree := client.IsAgree(line)
		if isAgree {
			break
		}

		messages = append(messages, &Message{
			Role:    "system",
			Content: commitMessage,
		})

		messages = append(messages, &Message{
			Role:    "user",
			Content: line,
		})
	}

	if err := commit(commitMessage); err != nil {
		fmt.Println(errors.WithMessage(err, "failed to commit"))
		os.Exit(1)
	}

	fmt.Println("Commit successfully")
}

// commit commits the changes
func commit(message string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("commit", "-m", message),
	)
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
