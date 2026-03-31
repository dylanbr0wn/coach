package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show installed agents and skills dashboard",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	agents, err := agent.DetectAgents("")
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	installed := agent.InstalledAgents(agents)
	if len(installed) == 0 {
		fmt.Println()
		fmt.Println(ui.Warn("No coding agents detected."))
		fmt.Println(ui.DimStyle.Render("  Coach looks for Claude Code, Cursor, Codex, and Copilot."))
		fmt.Println()
		return nil
	}

	coachDir := config.DefaultCoachDir()
	provenance, _ := registry.LoadProvenance(coachDir)
	provenanceMap := make(map[string]bool)
	for _, s := range provenance.Skills {
		provenanceMap[s.Name] = true
	}

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Detected Agents"))
	fmt.Println()

	var rows []ui.TableRow
	totalUnvetted := 0

	for _, a := range installed {
		skillCount := 0
		unvetted := 0
		skillNames := skill.ListSkillDirs(a.SkillDir)
		skillCount = len(skillNames)

		for _, name := range skillNames {
			if !provenanceMap[name] {
				unvetted++
			}
		}
		totalUnvetted += unvetted

		unvettedStr := fmt.Sprintf("%d", unvetted)
		if unvetted > 0 {
			unvettedStr = ui.WarningStyle.Render(unvettedStr)
		}

		rows = append(rows, ui.TableRow{
			Cells: []string{a.Config.Name, fmt.Sprintf("%d", skillCount), unvettedStr},
		})
	}

	fmt.Println(ui.RenderTable(
		[]string{"Agent", "Skills", "Unvetted"},
		rows,
	))

	fmt.Println()
	fmt.Println(ui.RenderStatusSummary(totalUnvetted, 0))
	fmt.Println()

	return nil
}
