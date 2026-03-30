package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/llm"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var (
	generatePrompt string
	generateGlobal bool
	generateLocal  bool
	generateCLI    string
)

var generateCmd = &cobra.Command{
	Use:   "generate <skill-name>",
	Short: "Author or refine a skill with LLM assistance",
	Long:  "Uses an LLM CLI (default: claude) to interactively author or refine a SKILL.md file. Pass --prompt for a quick single-shot edit.",
	Example: `  coach generate code-reviewer                              # Interactive: chat with LLM to author the skill
  coach generate code-reviewer -p "help review Go code"     # Single-shot: generate from a prompt
  coach generate new-skill -g                               # Create and author a new global skill
  coach generate my-skill --cli codex                       # Use a different LLM CLI`,
	Args: cobra.ExactArgs(1),
	RunE: runGenerate,
}

func init() {
	generateCmd.Flags().StringVarP(&generatePrompt, "prompt", "p", "", "Single-shot mode with inline instruction")
	generateCmd.Flags().BoolVarP(&generateGlobal, "global", "g", false, "Create/edit in global skills")
	generateCmd.Flags().BoolVarP(&generateLocal, "local", "l", false, "Create/edit in local project skills")
	generateCmd.Flags().StringVar(&generateCLI, "cli", "", "Override LLM CLI command")
	rootCmd.AddCommand(generateCmd)
}

func runGenerate(cmd *cobra.Command, args []string) error {
	name := args[0]

	// 1. Load config, determine CLI.
	coachDir := config.DefaultCoachDir()
	cfg, err := config.Load(coachDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cliCommand := cfg.LLMCli
	if generateCLI != "" {
		cliCommand = generateCLI
	}
	if cliCommand == "" {
		cliCommand = "claude"
	}

	// 2. FindCLI to verify it exists.
	cliPath, err := llm.FindCLI(cliCommand)
	if err != nil {
		return err
	}

	// 3. Resolve scope from flags.
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         workDir,
	}

	scope := resolve.ScopeAny
	if generateGlobal {
		scope = resolve.ScopeGlobal
	} else if generateLocal {
		scope = resolve.ScopeLocal
	}

	// 4. Try to resolve skill name.
	var skillPath string
	var skillDir string
	var existingContent string

	result, resolveErr := r.Resolve(name, scope)
	if resolveErr == nil {
		// Skill found — load existing content.
		skillPath = result.Path
		skillDir = result.Dir
		data, readErr := os.ReadFile(skillPath)
		if readErr != nil {
			return fmt.Errorf("reading existing skill: %w", readErr)
		}
		existingContent = string(data)
	} else {
		// Skill not found — create the directory and a minimal placeholder.
		targetDir := r.TargetDir(name, scope)
		if mkErr := os.MkdirAll(targetDir, 0o755); mkErr != nil {
			return fmt.Errorf("creating skill directory: %w", mkErr)
		}
		skillDir = targetDir
		skillPath = filepath.Join(skillDir, "SKILL.md")
		placeholder := fmt.Sprintf("---\nname: %s\ndescription: TODO — describe what this skill does\n---\n\n# %s\n\nTODO — add skill instructions here.\n", name, name)
		if writeErr := os.WriteFile(skillPath, []byte(placeholder), 0o644); writeErr != nil {
			return fmt.Errorf("creating placeholder skill: %w", writeErr)
		}
		existingContent = ""
	}

	// 5. Build system prompt.
	systemPrompt := llm.BuildSystemPrompt(existingContent, "")

	// 6. Dispatch to single-shot or interactive mode.
	if generatePrompt != "" {
		return runSingleShot(cliPath, systemPrompt, generatePrompt, skillPath, skillDir, name)
	}
	return runInteractive(cliPath, systemPrompt, skillDir, name)
}

func runSingleShot(cliPath, systemPrompt, userPrompt, skillPath, skillDir, skillName string) error {
	output, err := llm.RunSingleShot(cliPath, systemPrompt, userPrompt)
	if err != nil {
		return fmt.Errorf("LLM CLI error: %w", err)
	}

	result := strings.TrimSpace(string(output))
	if result == "" {
		return fmt.Errorf("LLM returned empty output")
	}

	fmt.Println()
	fmt.Println(result)
	fmt.Println()

	if !ui.PromptYesNo("Accept changes? [Y/n] ") {
		fmt.Println("Discarded.")
		return nil
	}

	if err := os.WriteFile(skillPath, []byte(result+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}
	fmt.Printf("  %s Skill updated: %s\n", ui.SuccessStyle.Render("✓"), skillName)
	fmt.Printf("  Path: %s\n", skillPath)
	fmt.Println()
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    %-36s   Validate all managed skills\n", ui.InfoStyle.Render("coach lint"))
	fmt.Printf("    %-36s   Distribute to your agents\n", ui.InfoStyle.Render("coach sync"))
	fmt.Println()

	return lintAfterGenerate(skillDir, skillName)
}

func runInteractive(cliPath, systemPrompt, skillDir, skillName string) error {
	fmt.Printf("%s Starting interactive session for skill %q. Type your instructions.\n", ui.InfoStyle.Render("→"), skillName)
	fmt.Printf("%s When done, the skill will be validated automatically.\n\n", ui.DimStyle.Render("hint"))

	if err := llm.RunInteractive(cliPath, systemPrompt); err != nil {
		return fmt.Errorf("LLM CLI exited with error: %w", err)
	}

	return lintAfterGenerate(skillDir, skillName)
}

func lintAfterGenerate(skillDir, skillName string) error {
	s, parseErr := skill.Parse(skillDir)
	if parseErr != nil {
		fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("✗ Parse error: %v", parseErr)))
		fmt.Printf("  Run %s to fix manually.\n", ui.InfoStyle.Render("coach edit "+skillName))
		return nil
	}

	issues := skill.Validate(s)
	if len(issues) > 0 {
		fmt.Println()
		for _, issue := range issues {
			fmt.Println(ui.ErrorStyle.Render("  ✗ " + issue))
		}
		fmt.Printf("\n  Run %s to fix manually.\n", ui.InfoStyle.Render("coach edit "+skillName))
		return nil
	}

	fmt.Printf("  %s %s validated successfully.\n", ui.SuccessStyle.Render("✓"), skillName)
	fmt.Println()
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    %-36s   Distribute to your agents\n", ui.InfoStyle.Render("coach sync"))
	fmt.Println()
	return nil
}
