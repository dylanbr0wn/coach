package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	Long:  "Distributes skills to configured coding agents by creating symlinks in each agent's skill directory.",
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
		return fmt.Errorf("no distribution targets configured. Run: coach config set distribute-to claude,cursor")
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

	if syncDryRun {
		fmt.Println(ui.HeadingStyle.Render("Dry run — would link:"))
		fmt.Println()
		for _, sk := range skills {
			scopeLabel := "global"
			if sk.Scope == resolve.ScopeLocal {
				scopeLabel = "local"
			}
			for _, t := range targets {
				linkPath := filepath.Join(t.SkillDir, sk.Name)
				fmt.Printf("  %s  %s → %s  %s\n",
					ui.SuccessStyle.Render("✓"),
					sk.Name,
					linkPath,
					ui.DimStyle.Render("("+scopeLabel+")"),
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

	return nil
}
