package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show managed skills and agent sync dashboard",
	Long: `Displays a two-section dashboard: managed skills across global and local
scopes, plus agent sync status showing which agents have skills installed.

See also: coach list (detailed per-agent skill listing), coach sync (distribute skills)`,
	Example: `  coach status                     # Show full dashboard
  coach list                       # Detailed per-agent skill listing
  coach sync                       # Distribute skills to agents`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	coachDir := config.DefaultCoachDir()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         workDir,
	}

	// --- Section 1: Managed Skills ---
	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Managed Skills"))

	globalSkills, _ := r.List(resolve.ScopeGlobal)
	localSkills, _ := r.List(resolve.ScopeLocal)

	globalDir := filepath.Join(coachDir, "skills") + "/"
	fmt.Printf("\n  %s\n", ui.DimStyle.Render(globalDir))
	if len(globalSkills) == 0 {
		fmt.Printf("    %s\n", ui.DimStyle.Render("(none)"))
	} else {
		for _, s := range globalSkills {
			desc := parseSkillDescription(s.Dir)
			fmt.Printf("    %s  %s  %s\n",
				ui.SuccessStyle.Render("\u2022"),
				ui.InfoStyle.Render(fmt.Sprintf("%-20s", s.Name)),
				ui.DimStyle.Render("\""+desc+"\""),
			)
		}
	}

	localDir := ".coach/skills/"
	fmt.Printf("\n  %s\n", ui.DimStyle.Render(localDir))
	if len(localSkills) == 0 {
		fmt.Printf("    %s\n", ui.DimStyle.Render("(none)"))
	} else {
		for _, s := range localSkills {
			desc := parseSkillDescription(s.Dir)
			fmt.Printf("    %s  %s  %s\n",
				ui.SuccessStyle.Render("\u2022"),
				ui.InfoStyle.Render(fmt.Sprintf("%-20s", s.Name)),
				ui.DimStyle.Render("\""+desc+"\""),
			)
		}
	}

	// --- Section 2: Agent Sync Status ---
	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Agent Status"))
	fmt.Println()

	agents, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	installed := agent.InstalledAgents(agents)
	if len(installed) == 0 {
		fmt.Println(ui.Warn("No coding agents detected.", ""))
		fmt.Println(ui.DimStyle.Render("  Coach looks for Claude Code, Cursor, Codex, and Copilot."))
		fmt.Println()
		return nil
	}

	provenance, _ := registry.LoadProvenance(coachDir)
	provenanceMap := make(map[string]bool)
	for _, s := range provenance.Skills {
		provenanceMap[s.Name] = true
	}

	for _, a := range installed {
		skillNames := skill.ListSkillDirs(a.SkillDir)
		skillCount := len(skillNames)

		unvetted := 0
		for _, name := range skillNames {
			if !provenanceMap[name] {
				unvetted++
			}
		}

		var parts []string
		parts = append(parts, fmt.Sprintf("%d skills synced", skillCount))
		if unvetted > 0 {
			parts = append(parts, ui.WarningStyle.Render(fmt.Sprintf("%d unvetted", unvetted)))
		}

		fmt.Printf("  %-18s %s\n", a.Config.Name, strings.Join(parts, ", "))
	}

	fmt.Println()
	return nil
}

// parseSkillDescription attempts to parse a SKILL.md and return a truncated description.
func parseSkillDescription(dir string) string {
	parsed, err := skill.Parse(dir)
	if err != nil {
		return "(parse error)"
	}
	desc := parsed.Description
	if utf8.RuneCountInString(desc) > 50 {
		desc = string([]rune(desc)[:47]) + "..."
	}
	return desc
}
