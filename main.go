package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/nguyenvanduocit/executils"
	"math/rand"
	"os"
	"os/exec"
	"strings"
)

var messages []*Message

func main() {

	autoCommit := false
	autoTag := false
	autoPush := false
	// parse args
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			showHelp()
			os.Exit(0)
		}

		if os.Args[1] == "-a" || os.Args[1] == "--auto" {
			autoCommit = true
		}

		if os.Args[1] == "-t" || os.Args[1] == "--tag" {
			autoTag = true
		}

		if os.Args[1] == "-p" || os.Args[1] == "--push" {
			autoPush = true
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
	defer func(client *GptClient) {
		printSuccess("Total token used: " + fmt.Sprint(client.totalToken))
	}(client)

	diff := ""
	var err error

	for {
		diff, err = getDiff()
		errGuard(client, err)

		if diff != "" {
			break
		}

		if !isDirty() {
			fmt.Println("Nothing to commit, working tree clean")
			os.Exit(0)
		}

		if autoCommit {
			errGuard(client, gitAdd())
			continue
		}

		shouldAutoStage, err := askForAutoStage(client)
		errGuard(client, err)

		if !shouldAutoStage {
			os.Exit(0)
		}

		errGuard(client, gitAdd())
	}

	commitMessage := ""
	messages = append(messages, &Message{
		Role:    "user",
		Content: "Write commit message for the following git diff: \n\n```" + diff + "\n\n```",
	})

	for {
		printNormal("Assistant: " + generateLoadingMessage())

		commitMessage, err = client.ChatComplete(context.Background(), messages)
		errGuard(client, err)

		deletePreviousLine(1)

		if commitMessage == "" {
			printNormal("Assistant: I don't know what to say about this diff, please give me a hint.")
			continue
		} else {
			printNormal("Assistant: " + commitMessage)
			messages = append(messages, &Message{
				Role:    "assistant",
				Content: commitMessage,
			})
		}

		if autoCommit {
			break
		}
		// Loop until the user response
		question, userResponse, err := askForUserResponse()
		errGuard(client, err)

		deletePreviousLine(2)
		fmt.Println("You: " + userResponse)

		isAgree, err := IsAgree(client, question, userResponse)
		errGuard(client, err)

		if isAgree {
			break
		}

		messages = append(messages, &Message{
			Role:    "user",
			Content: userResponse,
		})
	}

	if err := commit(commitMessage); err != nil {
		errGuard(client, err)
	}

	printSuccess("Assistant: " + getSuccessMessage())

	if autoTag {
		currentTag, err := getCurrentTag()
		errGuard(client, err)

		nextTag := "v0.0.0"
		if currentTag != "" {
			commits, err := listCommits(currentTag)
			fmt.Println(commits)
			errGuard(client, err)

			nextTag, err = getNextTag(client, commits, currentTag)
			errGuard(client, err)
		}

		printSuccess("Assistant: Next tag is " + nextTag)
		//errGuard(client, tag(nextTag))
		printSuccess("Assistant: New tag " + nextTag + " created")
	}

	if autoPush {
		errGuard(client, push())
		printSuccess("Assistant: Pushed to remote")
	}
}

func deletePreviousLine(numOfLine uint) {
	for i := 0; i < int(numOfLine); i++ {
		fmt.Print("\033[1A\033[K")
	}
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

func readUserInput(message string) (string, error) {
	fmt.Println("Assistant: " + message)
	fmt.Print("You: ")
	reader := bufio.NewReader(os.Stdin)
	userInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	userResponse := strings.TrimSpace(userInput)

	return userResponse, nil
}

func askForUserResponse() (string, string, error) {
	question := generateInteractiveMessage()
	userResponse, err := readUserInput(question)
	if err != nil {
		return "", "", err
	}

	if userResponse == "" {
		printWarning("Assistant: Please enter your response, say yes if you want to use the message or press Ctrl+C to exit")
		return askForUserResponse()
	}

	return question, userResponse, nil
}

// push current branch and current tag
func push() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "push", "--follow-tags")
	cmd.Dir = workingDir
	return cmd.Run()
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

// git log v1.1.3..HEAD --oneline
func listCommits(lastTag string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "log", lastTag+"..HEAD", "--oneline")
	cmd.Dir = workingDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func askForAutoStage(apiClient *GptClient) (bool, error) {
	question := "Your working tree is dirty, do you want me to stage the changes first?"
	userResponse, err := readUserInput(question)
	if err != nil {
		return false, err
	}

	if userResponse == "" {
		return askForAutoStage(apiClient)
	}

	isAgree, err := IsAgree(apiClient, question, userResponse)
	if err != nil {
		return false, err
	}

	return isAgree, nil
}

func getCurrentTag() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	cmd.Dir = workingDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func getNextTag(apiClient *GptClient, lastCommitMessage string, currentTag string) (string, error) {

	prompt := `Last tag: ` + currentTag + `
Conventional commit messages from the last tag to HEAD:
===
` + lastCommitMessage + `
===

What is the next tag?
Be careful, think step by step, but only response the tag name.`

	response, err := apiClient.ChatComplete(context.Background(), []*Message{
		{
			Role:    "system",
			Content: prompt,
		},
	})
	if err != nil {
		return "", err
	}

	return response, nil
}

func tag(tagName string) error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return executils.Run("git",
		executils.WithDir(workingDir),
		executils.WithArgs("tag", tagName),
	)
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

func errGuard(client *GptClient, err error) {
	if err == nil {
		return
	}

	if explain, explainErr := explainError(context.Background(), client, err); explainErr == nil {
		printError(err.Error() + ": " + explain)
		os.Exit(1)
	}

	printError(err.Error())
	os.Exit(1)
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
func IsAgree(c *GptClient, question, userResponse string) (bool, error) {
	message := []*Message{
		{
			Role:    "system",
			Content: "system: <generated commit message>\nassistant: " + question + "\nuser: " + userResponse + "\n\nDoes user mean yes or want to make change? (yes/change):",
		},
	}

	response, err := c.ChatComplete(context.Background(), message)
	if err != nil {
		return false, err
	}

	lowerResponse := strings.ToLower(response)

	return strings.HasPrefix(lowerResponse, "yes"), nil
}

var interactiveMessages = []string{
	"Is this commit message ok?",
	"Is it ok?",
	"Any changes?",
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
