package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/scanner"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/dylanbr0wn/coach/pkg"
)

var (
	installAgent string
	installCopy  bool
	installForce bool
	installList  bool
	installSkill string
)

var installCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install skills from GitHub or local path",
	Long:  "Fetches skills from a source, scans for security issues, and installs to detected agents.",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

func init() {
	installCmd.Flags().StringVar(&installAgent, "agent", "", "Install to specific agent only")
	installCmd.Flags().BoolVar(&installCopy, "copy", false, "Copy files instead of symlinking")
	installCmd.Flags().BoolVar(&installForce, "force", false, "Override critical security blocks")
	installCmd.Flags().BoolVar(&installList, "list", false, "List available skills without installing")
	installCmd.Flags().StringVar(&installSkill, "skill", "", "Install a specific skill from a multi-skill repo")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	src, err := registry.ParseSource(args[0])
	if err != nil {
		return err
	}

	var localPath string
	var sha string
	if spinErr := ui.WithSpinner(fmt.Sprintf("Fetching from %s", src.Raw), func() error {
		var fetchErr error
		localPath, sha, fetchErr = registry.FetchToCache(src)
		return fetchErr
	}); spinErr != nil {
		return fmt.Errorf("fetching source: %w", spinErr)
	}
	fmt.Println(ui.Success(fmt.Sprintf("Fetched %s", ui.DimStyle.Render(sha))))
	fmt.Println()

	skillPaths, err := registry.FindSkills(localPath)
	if err != nil {
		return fmt.Errorf("finding skills: %w", err)
	}
	if len(skillPaths) == 0 {
		return fmt.Errorf("no skills found in %s", src.Raw)
	}

	if installList {
		fmt.Println(ui.HeadingStyle.Render("  Available Skills"))
		fmt.Println()
		for _, sp := range skillPaths {
			s, err := skill.Parse(sp)
			if err != nil {
				fmt.Printf("  %s %s (parse error: %v)\n", ui.WarningStyle.Render("?"), filepath.Base(sp), err)
				continue
			}
			fmt.Printf("  %s %s — %s\n", ui.SuccessStyle.Render("•"), s.Name, ui.DimStyle.Render(s.Description))
		}
		fmt.Println()
		return nil
	}

	if installSkill != "" {
		var filtered []string
		for _, sp := range skillPaths {
			if filepath.Base(sp) == installSkill {
				filtered = append(filtered, sp)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("skill %q not found in source. Use --list to see available skills", installSkill)
		}
		skillPaths = filtered
	}

	agents, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}
	installedAgents := agent.InstalledAgents(agents)
	if len(installedAgents) == 0 {
		return fmt.Errorf("no coding agents detected")
	}

	if installAgent != "" {
		var filtered []pkg.DetectedAgent
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

	coachDir := config.DefaultCoachDir()
	rulesDir := filepath.Join(coachDir, "rules")
	db, err := rules.LoadPatterns(rulesDir)
	if err != nil {
		return fmt.Errorf("loading patterns: %w", err)
	}

	for _, sp := range skillPaths {
		s, err := skill.Parse(sp)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Skipping %s: %v", filepath.Base(sp), err), ""))
			continue
		}

		result := scanner.ScanSkill(s, db)
		fmt.Println(ui.RenderScanSummary(result))
		fmt.Println()

		if result.Risk == pkg.RiskCritical && !installForce {
			fmt.Println(ui.Error("Blocked", "use --force to override"))
			fmt.Println()
			continue
		}

		if result.Risk >= pkg.RiskMedium && !installForce {
			proceed := false
			if err := huh.NewConfirm().
				Title("Security warnings found. Install anyway?").
				Value(&proceed).
				Run(); err != nil {
				return fmt.Errorf("confirmation prompt: %w", err)
			}
			if !proceed {
				fmt.Println(ui.DimStyle.Render("  Skipped."))
				fmt.Println()
				continue
			}
		}

		var installedTo []string
		opts := registry.InstallOptions{Copy: installCopy}
		for _, a := range installedAgents {
			if err := registry.InstallSkill(sp, a.SkillDir, opts); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to install to %s: %v", a.Config.Name, err), ""))
				continue
			}
			installedTo = append(installedTo, a.Config.Name)
			fmt.Println(ui.Success(fmt.Sprintf("Installed to %s", a.Config.Name)))
		}

		if len(installedTo) > 0 {
			if err := registry.RecordInstall(coachDir, s.Name, src.Raw, sha, result.Score, installedTo); err != nil {
				fmt.Println(ui.Error(fmt.Sprintf("Failed to record install: %v", err), ""))
			}
		}
	}

	fmt.Println()
	return nil
}
