[![upvote](https://badge.vercel.app/producthunt/upvotes/ai-commit-2)](https://www.producthunt.com/posts/ai-commit-2)

[![Preview](./stuff/demo.gif)](https://youtu.be/7cVU3BuNpok)

[View Demo](https://youtu.be/7cVU3BuNpok)


# ai-commit

> No more headaches with commit messages.

AI-Commit is a command line tool that uses OpenAI's language generation capabilities to generate conventional commit messages for your Git repositories.

## Prerequisites

To use AI-Commit, you need to obtain an API key from OpenAI and set it as the value of the OPENAI_API_TOKEN environment variable.

Note: Using AI-Commit will result in charges from OpenAI for API usage, so be sure to understand their pricing model before use.

## Install

There are two ways to install AI-Commit:

### Use go

```bash
go install github.com/nguyenvanduocit/ai-commit@latest
```

### Prebuilt binaries

You can download prebuilt binaries for Linux, macOS, and Windows from the [releases page](https://github.com/nguyenvanduocit/ai-commit/releases)

## Usage

1. Stage the changes you want to commit in Git.
2. Run `ai-commit` command.
3. The tool will generate a commit message and print it to the console.
4. Now you can chat with the AI to adjust the commit message. Press ctrl + c to stop.
5. Finally, select the type of commit.

## Tip

It's recommended to make multiple small commits, commit more often.

Adjustment must be a list of adjectives to describe the commit message. Default adjustment is "short, simple, and clear"

## Todo

- [ ] Auto split changes in to multiple commits.
- [ ] Detect commit type.
- [ ] Auto tags??

## License

AI-Commit is released under the MIT license. See LICENSE for more information.

## Contributing

Contributions are welcome! Please read the [contribution guidelines](CONTRIBUTING.md) first.

[![update](./stuff/fhs.gif)](https://twitter.com/duocdev)
