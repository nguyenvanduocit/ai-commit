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

var messages []*Message

func main() {
	// prepare the arguments
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	systemPrompt := os.Getenv("AI_COMMIT_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = `You are a GitCommitGPT-4, You will help user to write commit message, commit message should be short (less than 100 chars), clean and meaningful. Only response the message.`
	}

	messages = []*Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	client := NewGptClient(apiKey)

	// prepare the diff
	diff, err := getDiff()
	if err != nil {
		if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
			printError(explain)
			os.Exit(1)
		}

		printError(err.Error())
		os.Exit(1)
	}

	if diff == "" {
		if !isDirty() {
			fmt.Println("Nothing to commit, working tree clean")
			os.Exit(0)
		}

		if shouldAutoStage := askForAutoStage(client); !shouldAutoStage {
			os.Exit(0)
		}

		if err := gitAdd(); err != nil {
			if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
				printError(explain)
				os.Exit(1)
			}

			printError(err.Error())
			os.Exit(1)
		}
	}

	commitMessage := ""
	// loop until the commit message is generated
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	messages = append(messages, &Message{
		Role:    "user",
		Content: "Write commit message for the following git diff: \n\n```" + diff + "\n\n```",
	})

	for {
		printNormal("Assistant: " + generateLoadingMessage())

		commitMessage, err = client.ChatComplete(ctx, messages)
		if err != nil {
			if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
				printError(explain)
				os.Exit(1)
			}

			printError(err.Error())
			os.Exit(1)
		}

		if commitMessage == "" {
			printNormal("Assistant: I don't know what to say about this diff, please give me a hint.")
		} else {
			printNormal("Assistant: " + commitMessage)
			messages = append(messages, &Message{
				Role:    "assistant",
				Content: commitMessage,
			})
		}

		// Loop until the user response
		userResponse, err := askForUserResponse()
		if err != nil {
			if explain, explainErr := explainError(ctx, client, err); explainErr == nil {
				printError(explain)
				os.Exit(1)
			}

			printError(err.Error())
			os.Exit(1)
		}

		if isAgree := IsAgree(client, userResponse); isAgree {
			break
		}

		messages = append(messages, &Message{
			Role:    "user",
			Content: userResponse,
		})
	}

	prefix := askForPrefix()
	commitMessage = joinPrefix(prefix, commitMessage)

	if err := commit(commitMessage); err != nil {
		printError("Assistant: failed to commit: " + err.Error())
		os.Exit(1)
	}

	printSuccess("Assistant: Commit successfully with message: " + commitMessage)
}

func askForUserResponse() (string, error) {
	fmt.Println("Assistant: " + generateInteractiveMessage())
	fmt.Print("You: ")

	reader := bufio.NewReader(os.Stdin)
	userResponse, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	userResponse = strings.TrimSpace(userResponse)

	if userResponse == "" {
		printWarning("Assistant: Please enter your response, say yes if you want to use the message or press Ctrl+C to exit")
		return askForUserResponse()
	}

	return userResponse, nil
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

func gitAdd() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("add", "."),
	)
}

func askForAutoStage(apiClient *GptClient) bool {
	fmt.Println("Assistant: Your working tree is dirty, but the stage is empty, do you want me to stage the changes first?")
	fmt.Print("You: ")
	reader := bufio.NewReader(os.Stdin)
	userRequest, err := reader.ReadString('\n')
	if err != nil {
		printError("failed to read user input: " + err.Error())
		os.Exit(1)
	}

	userRequest = strings.TrimSpace(userRequest)

	if userRequest == "" {
		return askForAutoStage(apiClient)
	}

	return IsAgree(apiClient, userRequest)
}

// rewrite func askForPrefix use recursion
func askForPrefix() string {
	fmt.Println("Assistant: Please enter the commit prefix, press enter to skip")
	fmt.Print("You: ")
	reader := bufio.NewReader(os.Stdin)
	prefix, err := reader.ReadString('\n')
	if err != nil {
		printError("failed to read user input: " + err.Error())
		os.Exit(1)
	}

	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return askForPrefix()
	}

	return prefix
}
func explainError(ctx context.Context, apiClient *GptClient, userError error) (string, error) {
	response, err := apiClient.ChatComplete(ctx, []*Message{
		{
			Role:    "system",
			Content: "You are a developer, explain the error to user: `" + userError.Error() + "`.",
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
	if err := executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("diff", "--cached", "--unified=0"),
		executils.WithStdOut(&out),
	); err != nil {
		return "", err
	}

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
	"please",
	"sure",
	"pls",
}

var disagreeWords = []string{
	"no",
	"n",
	"disagree",
	"nope",
	"no way",
	"nay",
	"nah",
	"never",
	"not",
	"don't",
}

// IsAgree returns true if the user agrees with the commit message
func IsAgree(c *GptClient, userResponse string) bool {
	for _, word := range agreeWords {
		if strings.HasPrefix(strings.ToLower(userResponse), word) {
			return true
		}
	}

	for _, word := range disagreeWords {
		if strings.HasPrefix(strings.ToLower(userResponse), word) {
			return false
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
