package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/nguyenvanduocit/executils"
	"math/rand"
	"os"
	"strings"
)

var messages []*Message

func main() {

	autoCommit := false
	// parse args
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			showHelp()
			os.Exit(0)
		}

		if os.Args[1] == "-a" || os.Args[1] == "--auto" {
			autoCommit = true
		}
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY is not set")
		os.Exit(1)
	}

	model := os.Getenv("AI_COMMIT_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	systemPrompt := os.Getenv("AI_COMMIT_SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = `You are a GitCommitGPT-4, You will help user to write conventional commit message, commit message should be short (less than 100 chars), clean and meaningful, be careful on commit type. Only response the message. If you can not write the message, response empty.`
	}

	messages = []*Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	client := NewGptClient(apiKey, model)

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

		if !autoCommit {
			if shouldAutoStage := askForAutoStage(client); !shouldAutoStage {
				os.Exit(0)
			}
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
	messages = append(messages, &Message{
		Role:    "user",
		Content: "Write commit message for the following git diff: \n\n```" + diff + "\n\n```",
	})

	for {
		printNormal("Assistant: " + generateLoadingMessage())

		commitMessage, err = client.ChatComplete(context.Background(), messages)
		if err != nil {
			if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
				printError(explain)
				os.Exit(1)
			}
			printError(err.Error())
			os.Exit(1)
		}

		fmt.Print("\033[1A\033[K")

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
			if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
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

	if err := commit(commitMessage); err != nil {
		printError("Assistant: failed to commit: " + err.Error())
		os.Exit(1)
	}

	printSuccess("Assistant: " + getSuccessMessage())
}

func showHelp() {
	fmt.Println("Usage: ai-commit [options]")

	fmt.Println("\nOptions:")
	fmt.Println("\t-h, --help\t Show help")
	fmt.Println("\t-a, --auto\t Auto stage all changes and commit, the changes could be split into multiple commits")

	fmt.Println("\nEnvironment variables:")
	fmt.Println("\tOPENAI_API_KEY\t OpenAI API key")
	fmt.Println("\tAI_COMMIT_MODEL\t OpenAI model, default is gpt-3.5-turbo")
	fmt.Println("\tAI_COMMIT_SYSTEM_PROMPT\t Default instruction for the assistant")

	fmt.Println("\nFollow me on twitter: @duocdev")
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

	// remove the 2nd last line
	fmt.Print("\033[1A\033[K")
	if userResponse == "" {
		printWarning("Assistant: Please enter your response, say yes if you want to use the message or press Ctrl+C to exit")
		return askForUserResponse()
	}

	return userResponse, nil
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

func explainError(ctx context.Context, apiClient *GptClient, userError error) (string, error) {
	response, err := apiClient.ChatComplete(ctx, []*Message{
		{
			Role:    "system",
			Content: "User run the cli tool ai-commit and got this error in their terminal, explain it: `" + userError.Error() + "`.",
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

var commitMessages = []string{
	"🚀 Blast off! Your commit has been launched into cyberspace!",
	"🎉 Woohoo! Your code change just joined the commit party!",
	"🍾 Pop the bubbly! That commit is now part of the code fam!",
	"🦄🌈 Your magical code change has been committed successfully!",
	"🤖 Beep boop! My AI circuits confirm your commit is in!",
	"🌟 Ta-da! Your commit has entered the code universe!",
	"🍪 Here's a cookie for your awesome commit! You did it!",
	"🏆 Achievement unlocked: Commit Master! Congrats!",
	"🎯 Bullseye! Your commit hit its mark in the codebase!",
	"🕺💃 Commit dance activated! Your change is in the mix!",
}

func getSuccessMessage() string {
	return commitMessages[rand.Intn(len(commitMessages))]
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
