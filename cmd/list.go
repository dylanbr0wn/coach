package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/dylanbr0wn/coach/internal/agent"
	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/registry"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/dylanbr0wn/coach/pkg"
	"github.com/spf13/cobra"
)

var (
	listAgent  string
	listFormat string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed skills per agent",
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listAgent, "agent", "", "Filter to a specific agent (e.g. claude-code, cursor)")
	listCmd.Flags().StringVar(&listFormat, "format", "table", "Output format: table or json")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	return runListWithHome(os.Stdout, "", config.DefaultCoachDir(), listAgent, listFormat)
}

// runListWithHome is the testable core. If home is empty, it uses os.UserHomeDir.
func runListWithHome(w io.Writer, home, coachDir, agentFilter, format string) error {
	if format != "table" && format != "json" {
		return fmt.Errorf("unsupported format %q (use \"table\" or \"json\")", format)
	}

	var agents []pkg.DetectedAgent
	var err error
	if home != "" {
		agents, err = agent.DetectAgentsInHome(home)
	} else {
		agents, err = agent.DetectAgents("")
	}
	if err != nil {
		return fmt.Errorf("detecting agents: %w", err)
	}

	installed := agent.InstalledAgents(agents)

	// Filter by agent key if requested
	if agentFilter != "" {
		var filtered []pkg.DetectedAgent
		for _, a := range installed {
			if a.Key == agentFilter {
				filtered = append(filtered, a)
			}
		}
		if len(filtered) == 0 {
			var keys []string
			for _, a := range installed {
				keys = append(keys, a.Key)
			}
			sort.Strings(keys)
			return fmt.Errorf("unknown agent %q (available: %v)", agentFilter, keys)
		}
		installed = filtered
	}

	if len(installed) == 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.WarningStyle.Render("  No agents detected."))
		fmt.Fprintln(w)
		return nil
	}

	provenance, err := registry.LoadProvenance(coachDir)
	if err != nil {
		provenance = &registry.InstalledSkills{}
	}
	provenanceMap := make(map[string]bool)
	for _, s := range provenance.Skills {
		provenanceMap[s.Name] = true
	}

	type skillInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Path        string `json:"path"`
		Vetted      bool   `json:"vetted"`
	}

	type agentGroup struct {
		Agent    string      `json:"agent"`
		SkillDir string      `json:"skill_dir"`
		Skills   []skillInfo `json:"skills"`
	}

	var groups []agentGroup

	for _, a := range installed {
		group := agentGroup{
			Agent:    a.Config.Name,
			SkillDir: a.SkillDir,
		}

		skillNames := skill.ListSkillDirs(a.SkillDir)
		for _, name := range skillNames {
			si := skillInfo{
				Name:   name,
				Path:   filepath.Join(a.SkillDir, name) + "/",
				Vetted: provenanceMap[name],
			}

			// Try to parse for description; gracefully handle errors
			parsed, parseErr := skill.Parse(filepath.Join(a.SkillDir, name))
			if parseErr == nil {
				si.Description = parsed.Description
			} else {
				si.Description = "(parse error)"
			}

			group.Skills = append(group.Skills, si)
		}

		groups = append(groups, group)
	}

	if format == "json" {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(groups)
	}

	// Table output
	for i, group := range groups {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, ui.HeadingStyle.Render("  "+group.Agent)+" "+ui.DimStyle.Render("("+group.SkillDir+")"))
		fmt.Fprintln(w)

		if len(group.Skills) == 0 {
			fmt.Fprintln(w, ui.DimStyle.Render("  No skills installed"))
			continue
		}

		var rows []ui.TableRow
		for _, s := range group.Skills {
			desc := s.Description
			if utf8.RuneCountInString(desc) > 40 {
				desc = string([]rune(desc)[:37]) + "..."
			}

			vetted := ui.SuccessStyle.Render("✓")
			if !s.Vetted {
				vetted = ui.WarningStyle.Render("✗")
			}

			rows = append(rows, ui.TableRow{
				Cells: []string{s.Name, desc, s.Path, vetted},
			})
		}

		fmt.Fprint(w, ui.RenderTable(
			[]string{"Name", "Description", "Path", "Vetted"},
			rows,
		))
	}

	fmt.Fprintln(w)
	return nil
}
