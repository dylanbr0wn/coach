package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/pipeline"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/types"
	"github.com/dylanbr0wn/coach/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	installAgent     string
	installCopy      bool
	installForce     bool
	installList      bool
	installSkill     string
	installScope     string
	installInstalled bool
	installYes       bool
)

var installCmd = &cobra.Command{
	Use:   "install [source]",
	Short: "Install skills from GitHub, local path, or audit installed skills",
	Long: `Batch-aware skill installation pipeline: discovers skills, evaluates them
(lint + security scan + quality checks), presents an interactive selection
table, and installs your choices to the configured scope and agents.

Supports GitHub repos, local directories, and auditing skills already
installed in agent directories.

See also: coach scan (security analysis), coach list (view installed skills), coach sync (distribute managed skills)`,
	Example: `  coach install owner/repo                      # Batch discover + interactive selection
  coach install ./local-skills                  # Install from local directory
  coach install owner/repo --yes                # Auto-approve all passing skills
  coach install owner/repo --list               # List available skills without installing
  coach install owner/repo --skill my-skill     # Install a specific skill from a repo
  coach install --installed                     # Audit untracked/modified skills in agent dirs
  coach install ./skills --scope local          # Install to project-local .coach/skills/
  coach install ./skills --copy                 # Copy instead of symlink`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installAgent, "agent", "", "Install to specific agent only")
	installCmd.Flags().BoolVar(&installCopy, "copy", false, "Copy files instead of symlinking")
	installCmd.Flags().BoolVar(&installForce, "force", false, "Override critical security blocks")
	installCmd.Flags().BoolVar(&installList, "list", false, "List available skills without installing")
	installCmd.Flags().StringVar(&installSkill, "skill", "", "Install a specific skill from a multi-skill repo")
	installCmd.Flags().StringVar(&installScope, "scope", "", "Install scope: global or local (default from config)")
	installCmd.Flags().BoolVar(&installInstalled, "installed", false, "Audit untracked/modified skills in agent directories")
	installCmd.Flags().BoolVar(&installYes, "yes", false, "Auto-approve all passing skills (skip interactive TUI)")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	if !installInstalled && len(args) == 0 {
		return fmt.Errorf("source is required (or use --installed to audit agent directories)")
	}

	coachDir := config.DefaultCoachDir()
	cfg, cfgErr := config.Load(coachDir)
	if cfgErr == nil && len(cfg.DistributeTo) == 0 {
		fmt.Fprintln(os.Stderr, ui.Warn("No agents configured for distribution",
			"Run 'coach setup' to get started, or set manually with 'coach config set distribute-to claude,cursor'"))
		fmt.Fprintln(os.Stderr)
	}

	// Detect agents.
	agents, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}
	installedAgents := agent.InstalledAgents(agents)
	if len(installedAgents) == 0 {
		return fmt.Errorf("no coding agents detected")
	}
	if installAgent != "" {
		var filtered []types.DetectedAgent
		for _, a := range installedAgents {
			if a.Config.Name == installAgent {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("agent %q not found or not installed", installAgent)
		}
		installedAgents = filtered
	}

	// Load provenance for installed audit.
	provenance, _ := registry.LoadProvenance(coachDir)

	// --- Stage 1: Discover ---
	var candidates []pipeline.SkillCandidate
	sourceLabel := ""

	if installInstalled {
		sourceLabel = "installed agent skills"
		candidates, err = pipeline.Discover(nil, true, installedAgents, provenance)
		if err != nil {
			return fmt.Errorf("discovering installed skills: %w", err)
		}
	} else {
		src, parseErr := registry.ParseSource(args[0])
		if parseErr != nil {
			return parseErr
		}
		sourceLabel = src.Raw
		if spinErr := ui.WithSpinner(fmt.Sprintf("Fetching from %s", src.Raw), func() error {
			candidates, err = pipeline.Discover(src, false, nil, nil)
			return err
		}); spinErr != nil {
			return fmt.Errorf("discovering skills: %w", spinErr)
		}
	}

	if len(candidates) == 0 {
		fmt.Println(ui.Warn("No skills found", fmt.Sprintf("source: %s", sourceLabel)))
		return nil
	}

	// Filter by --skill if specified.
	if installSkill != "" {
		var filtered []pipeline.SkillCandidate
		for _, c := range candidates {
			if filepath.Base(c.Path) == installSkill {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("skill %q not found in source. Use --list to see available skills", installSkill)
		}
		candidates = filtered
	}

	// --list: print discovered skills and exit.
	if installList {
		fmt.Println(ui.HeadingStyle.Render("  Available Skills"))
		fmt.Println()
		for _, c := range candidates {
			s, parseErr := skill.Parse(c.Path)
			if parseErr != nil {
				fmt.Printf("  %s %s (parse error: %v)\n", ui.WarningStyle.Render("?"), filepath.Base(c.Path), parseErr)
				continue
			}
			fmt.Printf("  %s %s — %s\n", ui.SuccessStyle.Render("•"), s.Name, ui.DimStyle.Render(s.Description))
		}
		fmt.Println()
		return nil
	}

	// --- Stage 2: Evaluate ---
	rulesDir := filepath.Join(coachDir, "rules")
	db, err := rules.LoadPatterns(rulesDir)
	if err != nil {
		return fmt.Errorf("loading patterns: %w", err)
	}

	var vetted []pipeline.VettedSkill

	if installYes {
		// Non-interactive: evaluate with printed progress.
		vetted, err = pipeline.Evaluate(candidates, db, installForce, func(current, total int, name string) {
			fmt.Fprintf(os.Stderr, "\r  Evaluating %d/%d: %s", current, total, name)
		})
		fmt.Fprintln(os.Stderr) // newline after progress
		if err != nil {
			return fmt.Errorf("evaluating skills: %w", err)
		}
	} else {
		// Interactive: run TUI with progress bar + selection table.
		model := ui.NewInstallModel(sourceLabel, installForce)

		// Run evaluation synchronously first (TUI progress is a stretch goal;
		// for now we use a spinner then show the selection table).
		if spinErr := ui.WithSpinner(
			fmt.Sprintf("Evaluating %d skills", len(candidates)),
			func() error {
				var evalErr error
				vetted, evalErr = pipeline.Evaluate(candidates, db, installForce, nil)
				return evalErr
			},
		); spinErr != nil {
			return fmt.Errorf("evaluating skills: %w", spinErr)
		}

		model.SetEvalDone(vetted)

		p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}

		m := finalModel.(*ui.InstallModel)
		if m.Cancelled() {
			fmt.Println(ui.DimStyle.Render("  Install cancelled."))
			return nil
		}
		if m.Err() != nil {
			return m.Err()
		}

		selected := m.Selected()
		if len(selected) == 0 {
			fmt.Println(ui.DimStyle.Render("  No skills selected."))
			return nil
		}
		vetted = selected
	}

	// In --yes mode, auto-select all passing skills.
	if installYes {
		var autoSelected []pipeline.VettedSkill
		for _, v := range vetted {
			if v.Selectable {
				autoSelected = append(autoSelected, v)
			}
		}
		if len(autoSelected) == 0 {
			fmt.Println(ui.Warn("No skills passed evaluation", "use --force to override critical blocks"))
			return nil
		}
		vetted = autoSelected
	}

	// --- Scope selection ---
	scope := installScope
	if scope == "" {
		if cfg != nil && cfg.DefaultScope != "" {
			scope = cfg.DefaultScope
		} else {
			scope = "global"
		}
	}

	if !installYes {
		var scopeChoice string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title(fmt.Sprintf("Install %d skills to:", len(vetted))).
					Options(
						huh.NewOption("global (~/.coach/skills/)", "global"),
						huh.NewOption("local (.coach/skills/)", "local"),
					).
					Value(&scopeChoice),
			),
		).Run(); err != nil {
			return fmt.Errorf("scope selection: %w", err)
		}
		scope = scopeChoice
	}

	// --- Stage 4: Commit ---
	opts := pipeline.InstallOptions{
		Copy:   installCopy,
		Force:  installForce,
		Scope:  scope,
		Agents: installedAgents,
	}

	results, err := pipeline.Commit(vetted, coachDir, opts)
	if err != nil {
		return fmt.Errorf("installing skills: %w", err)
	}

	// --- Summary ---
	scopeLabel := "global"
	if scope == "local" {
		scopeLabel = "local"
	}
	fmt.Println()
	fmt.Println(ui.Success(fmt.Sprintf("Installed %d skills (%s)", len(results), scopeLabel)))
	fmt.Println()
	for _, r := range results {
		if r.Err != nil {
			fmt.Printf("  %s %s — %s\n", ui.ErrorStyle.Render("✗"), r.Name, r.Err)
		} else if len(r.Agents) > 0 {
			agentList := ""
			for i, a := range r.Agents {
				if i > 0 {
					agentList += ", "
				}
				agentList += a
			}
			fmt.Printf("  %s → %s\n", ui.Success(r.Name), ui.DimStyle.Render(agentList))
		} else {
			fmt.Printf("  %s\n", ui.Success(r.Name))
		}
	}
	fmt.Println()
	fmt.Println(ui.NextStep("list", "see all installed skills"))

	return nil
}
