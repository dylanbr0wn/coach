package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive first-run configuration wizard",
	Long:  "Detects installed agents, configures distribution targets and LLM preferences, and creates required directories.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Step 1 — Agent selection
	detected, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	installed := agent.InstalledAgents(detected)
	if len(installed) == 0 {
		fmt.Fprintln(os.Stderr, ui.Error("No coding agents detected", ""))
		fmt.Fprintln(os.Stderr, ui.DimStyle.Render("  Coach supports Claude Code, Cursor, Codex, and Copilot."))
		fmt.Fprintln(os.Stderr, ui.DimStyle.Render("  Install a supported agent, then re-run 'coach setup'."))
		return fmt.Errorf("no coding agents detected")
	}

	var agentOptions []huh.Option[string]
	for _, a := range installed {
		label := fmt.Sprintf("%s (%s)", a.Config.Name, a.SkillDir)
		agentOptions = append(agentOptions, huh.NewOption(label, a.Key))
	}

	var selectedAgents []string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which agents should receive your skills?").
				Options(agentOptions...).
				Value(&selectedAgents),
		),
	).Run()
	if err != nil {
		return err
	}

	if len(selectedAgents) == 0 {
		return fmt.Errorf("no agents selected")
	}

	// Step 2 — LLM CLI preference
	var llmChoice string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which LLM CLI do you use for generation?").
				Options(
					huh.NewOption("claude", "claude"),
					huh.NewOption("codex", "codex"),
					huh.NewOption("gemini", "gemini"),
					huh.NewOption("other", "other"),
					huh.NewOption("skip", "skip"),
				).
				Value(&llmChoice),
		),
	).Run()
	if err != nil {
		return err
	}

	var llmCli string
	if llmChoice == "other" {
		var customCli string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Enter your LLM CLI command").
					Value(&customCli),
			),
		).Run()
		if err != nil {
			return err
		}
		llmCli = strings.TrimSpace(customCli)
	} else if llmChoice != "skip" {
		llmCli = llmChoice
	}

	// Step 3 — Directory creation
	coachDir := config.DefaultCoachDir()
	if err := config.EnsureCoachDir(coachDir); err != nil {
		return fmt.Errorf("creating coach directories: %w", err)
	}

	skillsDir := filepath.Join(coachDir, "skills")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("creating skills directory: %w", err)
	}

	// Create agent skill directories for selected agents.
	for _, a := range detected {
		for _, sel := range selectedAgents {
			if a.Key == sel {
				if err := os.MkdirAll(a.SkillDir, 0o755); err != nil {
					return fmt.Errorf("creating skill directory for %s: %w", a.Config.Name, err)
				}
			}
		}
	}

	// Save config
	cfg, err := config.Load(coachDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	cfg.DistributeTo = selectedAgents
	if llmCli != "" {
		cfg.LLMCli = llmCli
	}

	if err := config.Save(coachDir, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Step 4 — Confirmation
	fmt.Println()
	fmt.Fprintln(os.Stderr, ui.Success("Coach configured successfully"))
	fmt.Fprintf(os.Stderr, "  %s %s\n", ui.DimStyle.Render("Agents:"), strings.Join(selectedAgents, ", "))
	if llmCli != "" {
		fmt.Fprintf(os.Stderr, "  %s %s\n", ui.DimStyle.Render("LLM CLI:"), llmCli)
	}
	fmt.Fprintf(os.Stderr, "  %s %s\n", ui.DimStyle.Render("Skills:"), skillsDir)

	fmt.Fprintln(os.Stderr, ui.NextStep("init skill", "scaffold your first skill"))

	return nil
}
