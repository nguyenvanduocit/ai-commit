# ai-commit

> No more headaches with commit messages.

AI-Commit is a command line tool that uses OpenAI's language generation capabilities to generate conventional commit messages for your Git repositories.

## Prerequisites

To use AI-Commit, you need to obtain an API key from OpenAI and set it as the value of the OPENAI_API_TOKEN environment variable.

Note: Using AI-Commit will result in charges from OpenAI for API usage, so be sure to understand their pricing model before use.

## Install

### Use go

```bash
go install github.com/nguyenvanduocit/ai-commit
```

### Prebuilt binaries

You can download prebuilt binaries for Linux, macOS, and Windows from the [releases page]()

## Usage

1. Stage the changes you want to commit in Git.
2. Run `ai-commit` command.
3. The tool will generate a commit message and print it to the console.

If you want the tool auto create commit, you can use `--commit` flag.

## Tip

It's recommended to make multiple small commits, commit more often.

## Todo

- [ ] Auto split changes in to multiple commits.

## License

AI-Commit is released under the MIT license. See LICENSE for more information.

## Contributing

Contributions are welcome! Please read the [contribution guidelines](CONTRIBUTING.md) first.
