package llm

import (
	"fmt"
	"os/exec"
)

// FindCLI verifies that the given CLI command exists in PATH and returns its full path.
// Returns a helpful error message if the command is not found.
func FindCLI(command string) (string, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("%s CLI not found. Install it or configure a different CLI: coach config set llm-cli %s", command, command)
	}
	return path, nil
}

// BuildSingleShotArgs returns the CLI arguments for a single-shot (non-interactive) invocation.
// The returned slice is: ["--print", "-p", userPrompt, "--system-prompt", systemPrompt]
func BuildSingleShotArgs(cliName, systemPrompt, userPrompt string) []string {
	return []string{"--print", "-p", userPrompt, "--system-prompt", systemPrompt}
}

// BuildInteractiveArgs returns the CLI arguments for an interactive session.
// The returned slice is: ["--system-prompt", systemPrompt]
func BuildInteractiveArgs(cliName, systemPrompt string) []string {
	return []string{"--system-prompt", systemPrompt}
}
