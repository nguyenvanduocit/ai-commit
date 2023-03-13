package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/nguyenvanduocit/executils"
	"math/rand"
	"os"
	"strings"
	"time"
)

var messages = []*Message{
	{
		Role:    "system",
		Content: `You are a developer, who are very good at write git commit, write commit message for this diff, only response the message:`,
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
		explain, err := explainError(context.Background(), client, err)
		if err != nil {
			printError("failed to explain error: " + err.Error())
			os.Exit(1)
		}
		printError(explain)
		os.Exit(1)
	}

	if diff == "" {
		if isDirty() {
			fmt.Println("Please stage your changes and try again")
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

		printNormal("Assistant: " + generateLoadingMessage())
		commitMessage, err = client.ChatComplete(ctx, messages)
		if err != nil {
			explain, err := explainError(ctx, client, err)
			if err != nil {
				printError("failed to explain error: " + err.Error())
				os.Exit(1)
			}
			printError(explain)
			os.Exit(1)
		}

		if commitMessage == "" {
			printNormal("No commit message generated, please try again")
		} else {
			printNormal("Assistant: " + commitMessage)
		}

		userRequest := ""
		for {
			fmt.Println("Assistant: " + generateInteractiveMessage())
			fmt.Print("You: ")
			reader := bufio.NewReader(os.Stdin)
			userRequest, err = reader.ReadString('\n')
			if err != nil {
				explain, err := explainError(ctx, client, err)
				if err != nil {
					printError("failed to explain error: " + err.Error())
					os.Exit(1)
				}
				printError(explain)
				os.Exit(1)
			}

			userRequest = strings.TrimSpace(userRequest)

			if userRequest == "" {
				printWarning("Please enter your response, say yes if you want to use the message or press Ctrl+C to exit")
				continue
			}

			break
		}

		if isAgree := IsAgree(client, userRequest); isAgree {
			break
		}

		if commitMessage != "" {
			messages = append(messages, &Message{
				Role:    "assistant",
				Content: commitMessage,
			})
		} else {
			// replace the last message
			messages[len(messages)-1].Content = userRequest
		}
	}

	prefix := askForPrefix()
	commitMessage = joinPrefix(prefix, commitMessage)

	if err := commit(commitMessage); err != nil {
		printError("failed to commit: " + err.Error())
		os.Exit(1)
	}

	printSuccess("Commit successfully with message: " + commitMessage)
}

func joinPrefix(prefix string, message string) string {

	if prefix == "" {
		return message
	}

	messageParts := strings.Split(message, ":")
	if len(messageParts) == 2 {
		message = strings.TrimSpace(messageParts[1])
	}

	return prefix + ": " + message
}

func askForPrefix() string {
	prefix := ""
	var err error
	for {
		fmt.Println("Assistant: Please enter the commit prefix, press enter to skip")
		fmt.Print("You: ")
		reader := bufio.NewReader(os.Stdin)
		prefix, err = reader.ReadString('\n')
		if err != nil {
			printError("failed to read user input: " + err.Error())
			os.Exit(1)
		}

		prefix = strings.TrimSpace(prefix)
		break
	}

	return prefix
}

func explainError(ctx context.Context, apiClient *GptClient, userError error) (string, error) {
	response, err := apiClient.ChatComplete(ctx, []*Message{
		{
			Role:    "system",
			Content: "You are a developer, explain the error to user: `" + userError.Error() + "`, only response the message:",
		},
	})
	if err != nil {
		return "", userError
	}

	return response, nil
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

var interactiveMessages = []string{
	"Is this commit message ok?",
	"Is it ok?",
	"Changes?",
	"Any changes?",
	"Looks good?",
}

func generateInteractiveMessage() string {
	return interactiveMessages[rand.Intn(len(interactiveMessages))]
}

var loadingMessages = []string{
	"Let me think a bit ...",
	"I'm thinking ...",
	"Thinking ...",
	"Wait a second ...",
	"Lets see ...",
	"Generating ...",
	"What we have here ...",
	"Looking at the changes ...",
	"Oh, awesome codes ...",
	"Oh no, what's this ...",
}

func generateLoadingMessage() string {
	return loadingMessages[rand.Intn(len(loadingMessages))]
}
