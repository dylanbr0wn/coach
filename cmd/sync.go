package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/distribute"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var (
	syncGlobal bool
	syncLocal  bool
	syncDryRun bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Symlink managed skills into configured agent directories",
	Long: `Distributes skills to configured coding agents by creating symlinks in each
agent's skill directory. If no targets are configured, prompts interactively.

See also: coach status (dashboard overview), coach list (view installed skills)`,
	Example: `  coach sync                # Symlink all skills to configured agents
  coach sync --dry-run      # Preview what would be linked
  coach sync -g             # Sync global skills only`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVarP(&syncGlobal, "global", "g", false, "Sync global skills only")
	syncCmd.Flags().BoolVarP(&syncLocal, "local", "l", false, "Sync local skills only")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "Preview without making changes")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	coachDir := config.DefaultCoachDir()
	cfg, err := config.Load(coachDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.DistributeTo) == 0 {
		fmt.Fprintln(os.Stderr, ui.Warn("No agents configured for distribution",
			"Run 'coach setup' to get started, or set manually with 'coach config set distribute-to claude,cursor'"))
		fmt.Fprintln(os.Stderr)

		detected, detectErr := agent.DetectAgents("")
		if detectErr != nil {
			return fmt.Errorf("detecting agents: %w", detectErr)
		}

		var agentOptions []huh.Option[string]
		for _, a := range detected {
			label := fmt.Sprintf("%s (%s)", a.Config.Name, a.SkillDir)
			agentOptions = append(agentOptions, huh.NewOption(label, a.Key))
		}

		if len(agentOptions) == 0 {
			return fmt.Errorf("no agents detected on this system")
		}

		var selected []string
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("No distribution targets configured. Which agents should receive your skills?").
					Options(agentOptions...).
					Value(&selected),
			),
		).Run()
		if err != nil {
			return err
		}

		if len(selected) == 0 {
			return fmt.Errorf("no agents selected")
		}

		cfg.DistributeTo = selected
		if saveErr := config.Save(coachDir, cfg); saveErr != nil {
			return fmt.Errorf("saving config: %w", saveErr)
		}
		fmt.Println(ui.Success(fmt.Sprintf("Saved distribution targets: %s", strings.Join(selected, ", "))))
		fmt.Println()
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         workDir,
	}

	scope := resolve.ScopeAny
	if syncGlobal {
		scope = resolve.ScopeGlobal
	} else if syncLocal {
		scope = resolve.ScopeLocal
	}

	skills, err := r.List(scope)
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println("No managed skills found.")
		return nil
	}

	detected, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	targets := distribute.FilterAgentsByNames(detected, cfg.DistributeTo)
	if len(targets) == 0 {
		return fmt.Errorf("no configured agents detected (looking for: %s)", strings.Join(cfg.DistributeTo, ", "))
	}

	// Ensure agent skill directories exist for configured targets.
	// This handles the case where the agent is installed but its
	// skill directory hasn't been created yet.
	for i := range targets {
		if !targets[i].Installed {
			if err := os.MkdirAll(targets[i].SkillDir, 0o755); err != nil {
				return fmt.Errorf("creating skill directory for %s: %w", targets[i].Config.Name, err)
			}
			targets[i].Installed = true
		}
	}

	if syncDryRun {
		fmt.Println(ui.HeadingStyle.Render("Dry run — would link:"))
		fmt.Println()
		for _, sk := range skills {
			for _, t := range targets {
				if !t.Installed {
					fmt.Printf("  %s  %s → %s %s\n",
						ui.DimStyle.Render("-"),
						sk.Name,
						t.Config.Name,
						ui.DimStyle.Render("(skipped — not installed)"),
					)
					continue
				}
				fmt.Printf("  %s  %s → %s\n",
					ui.SuccessStyle.Render("✓"),
					sk.Name,
					t.Config.Name,
				)
			}
		}
		fmt.Println()
		fmt.Printf("%s  %d skill(s) → %d agent(s)\n",
			ui.DimStyle.Render("(dry run)"),
			len(skills),
			len(targets),
		)
		return nil
	}

	totals := map[distribute.Status]int{
		distribute.StatusCreated:  0,
		distribute.StatusUpdated:  0,
		distribute.StatusUpToDate: 0,
		distribute.StatusSkipped:  0,
	}

	for _, sk := range skills {
		results, err := distribute.Distribute(sk.Dir, sk.Name, targets)
		if err != nil {
			return fmt.Errorf("distributing %s: %w", sk.Name, err)
		}

		for _, res := range results {
			totals[res.Status]++
			switch res.Status {
			case distribute.StatusCreated:
				fmt.Printf("  %s  %s → %s\n",
					ui.SuccessStyle.Render("✓"),
					sk.Name,
					res.Agent,
				)
			case distribute.StatusUpdated:
				fmt.Printf("  %s  %s → %s %s\n",
					ui.SuccessStyle.Render("✓"),
					sk.Name,
					res.Agent,
					ui.DimStyle.Render("(updated)"),
				)
			case distribute.StatusUpToDate:
				fmt.Printf("  %s  %s → %s %s\n",
					ui.DimStyle.Render("·"),
					sk.Name,
					res.Agent,
					ui.DimStyle.Render("(up-to-date)"),
				)
			case distribute.StatusSkipped:
				fmt.Printf("  %s  %s → %s %s\n",
					ui.DimStyle.Render("-"),
					sk.Name,
					res.Agent,
					ui.DimStyle.Render("(skipped — not installed)"),
				)
			}
		}
	}

	fmt.Println()
	fmt.Printf("%s  created: %d  updated: %d  up-to-date: %d  skipped: %d\n",
		ui.HeadingStyle.Render("Sync complete."),
		totals[distribute.StatusCreated],
		totals[distribute.StatusUpdated],
		totals[distribute.StatusUpToDate],
		totals[distribute.StatusSkipped],
	)

	fmt.Fprintln(os.Stderr, ui.NextStep("status", "verify everything looks right"))

	return nil
}
