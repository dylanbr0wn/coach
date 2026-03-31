package llm

import (
	"fmt"
	"os"
	"os/exec"
)

// CommandRunner abstracts command execution for testability.
type CommandRunner interface {
	// Output runs the command and returns its stdout.
	Output(name string, args ...string) ([]byte, error)
	// Run runs the command with stdin/stdout/stderr connected.
	Run(name string, args ...string) error
}

// ExecRunner implements CommandRunner using os/exec.
type ExecRunner struct{}

func (ExecRunner) Output(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func (ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DefaultRunner is the package-level CommandRunner used by RunSingleShot and RunInteractive.
var DefaultRunner CommandRunner = ExecRunner{}

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
func BuildSingleShotArgs(systemPrompt, userPrompt string) []string {
	return []string{"--print", "-p", userPrompt, "--system-prompt", systemPrompt}
}

// BuildInteractiveArgs returns the CLI arguments for an interactive session.
func BuildInteractiveArgs(systemPrompt string) []string {
	return []string{"--system-prompt", systemPrompt}
}

// RunSingleShot executes the LLM CLI in single-shot mode and returns the captured stdout.
func RunSingleShot(cliPath, systemPrompt, userPrompt string) ([]byte, error) {
	args := BuildSingleShotArgs(systemPrompt, userPrompt)
	return DefaultRunner.Output(cliPath, args...)
}

// RunInteractive executes the LLM CLI in interactive mode, connecting stdin/stdout/stderr.
func RunInteractive(cliPath, systemPrompt string) error {
	args := BuildInteractiveArgs(systemPrompt)
	return DefaultRunner.Run(cliPath, args...)
}
