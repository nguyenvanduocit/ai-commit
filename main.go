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
		Content: `You are a senior developer, you are writing commit message for this diff, make it short, but meaningful, only response the message`,
	},
}

func main() {

	// prepare the arguments
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	client := NewGptClient(apiKey)

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

	commitMessage := ""
	for {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
		messages = append(messages, &Message{
			Role:    "user",
			Content: diff,
		})

		commitMessage, err = client.ChatComplete(ctx, messages)
		if err != nil {
			printError("failed to generate commit message: " + err.Error())
			os.Exit(1)
		}

		if commitMessage == "" {
			commitMessage = "I can not understand your message, please try again"
		}

		printNormal(commitMessage)

		userRequest := ""
		for {
			fmt.Print("User: ")
			reader := bufio.NewReader(os.Stdin)
			userRequest, err = reader.ReadString('\n')
			if err != nil {
				printError("failed to read user input: " + err.Error())
				os.Exit(1)
			}

			userRequest = strings.TrimSpace(userRequest)

			if userRequest == "" {
				printWarning("Please enter your response")
				continue
			}

			break
		}

		if isAgree := IsAgree(client, userRequest); isAgree {
			break
		}

		if commitMessage != "" {
			messages = append(messages, &Message{
				Role:    "system",
				Content: commitMessage,
			})
		} else {
			// replace the last message
			messages[len(messages)-1].Content = userRequest
		}
	}

	// ask for prefix
	prefix := askForPrefix()
	commitMessage = prefix + ": " + commitMessage

	if err := commit(commitMessage); err != nil {
		printError("failed to commit: " + err.Error())
		os.Exit(1)
	}

	printSuccess("Commit successfully with message: " + commitMessage)
}

func askForPrefix() string {
	prefix := ""
	for {
		fmt.Print("Commit prefix: ")
		reader := bufio.NewReader(os.Stdin)
		prefix, err := reader.ReadString('\n')
		if err != nil {
			printError("failed to read user input: " + err.Error())
			os.Exit(1)
		}

		prefix = strings.TrimSpace(prefix)

		if prefix == "" {
			printWarning("Please enter your commit prefix")
			continue
		}

		if !isPrefixValid(prefix) {
			printWarning("Invalid commit prefix, please try again")
			continue
		}

		break
	}

	return prefix
}

var prefixes = []string{
	"feat",
	"fix",
	"docs",
	"style",
	"refactor",
	"perf",
	"test",
	"chore",
	"revert",
	"build",
}

func isPrefixValid(prefix string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(prefix, p) {
			return true
		}
	}

	return false
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

var agreeWords = []string{
	"yes",
	"y",
	"ok",
	"okay",
	"agree",
}

// IsAgree returns true if the user agrees with the commit message
func IsAgree(c *GptClient, userResponse string) bool {
	for _, word := range agreeWords {
		if strings.HasPrefix(strings.ToLower(userResponse), word) {
			return true
		}
	}

	message := []*Message{
		{
			Role:    "user",
			Content: "only response with \"change request\" or \"agreement\"; the following message is a change request or agreement: " + userResponse,
		},
	}

	response, err := c.ChatComplete(context.Background(), message)
	if err != nil {
		return false
	}

	lowerResponse := strings.ToLower(response)

	return strings.HasPrefix(lowerResponse, "agreement")
}
