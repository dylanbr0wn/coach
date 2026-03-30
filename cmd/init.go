package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new skill, hook, or agent config",
}

var initSkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Scaffold a new agent skill with SKILL.md",
	RunE:  runInitSkill,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initSkillCmd)
}

func runInitSkill(cmd *cobra.Command, args []string) error {
	var (
		name           string
		description    string
		license        string
		tools          string
		includeTests   bool
		includeScripts bool
		includeRefs    bool
	)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Skill name").
				Description("Lowercase, hyphens only (e.g. 'my-skill')").
				Value(&name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					for _, c := range s {
						if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
							return fmt.Errorf("name must be lowercase alphanumeric with hyphens only")
						}
					}
					if len(s) > 64 {
						return fmt.Errorf("name must be 64 characters or less")
					}
					return nil
				}),

			huh.NewText().
				Title("Description").
				Description("What does this skill do? (max 1024 chars)").
				Value(&description).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("description is required")
					}
					if len(s) > 1024 {
						return fmt.Errorf("description must be 1024 characters or less")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("License").
				Options(
					huh.NewOption("MIT", "MIT"),
					huh.NewOption("Apache-2.0", "Apache-2.0"),
					huh.NewOption("ISC", "ISC"),
					huh.NewOption("None", ""),
				).
				Value(&license),

			huh.NewInput().
				Title("Allowed tools (comma-separated, or empty for no restrictions)").
				Description("e.g. Read,Write,Bash").
				Value(&tools),
		),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Include tests/ directory?").
				Value(&includeTests),

			huh.NewConfirm().
				Title("Include scripts/ directory?").
				Value(&includeScripts),

			huh.NewConfirm().
				Title("Include references/ directory?").
				Value(&includeRefs),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	dir := name
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	var toolsList []string
	if tools != "" {
		for _, t := range strings.Split(tools, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				toolsList = append(toolsList, t)
			}
		}
	}

	skillContent := generateSkillMD(name, description, license, toolsList)
	skillPath := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(skillPath, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	if includeTests {
		os.MkdirAll(filepath.Join(dir, "tests"), 0755)
	}
	if includeScripts {
		os.MkdirAll(filepath.Join(dir, "scripts"), 0755)
	}
	if includeRefs {
		os.MkdirAll(filepath.Join(dir, "references"), 0755)
	}

	successMsg := lipgloss.NewStyle().Bold(true).Foreground(ui.Green).Render("Skill scaffolded!")
	fmt.Printf("\n%s\n\n", successMsg)
	fmt.Printf("  %s %s\n", ui.LabelStyle.Render("Directory:"), dir)
	fmt.Printf("  %s %s\n", ui.LabelStyle.Render("SKILL.md:"), skillPath)
	fmt.Printf("\n  Run %s to check for issues.\n\n", lipgloss.NewStyle().Bold(true).Render("coach lint "+dir))

	return nil
}

func generateSkillMD(name, description, license string, tools []string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	sb.WriteString(fmt.Sprintf("description: %s\n", description))
	if license != "" {
		sb.WriteString(fmt.Sprintf("license: %s\n", license))
	}
	if len(tools) > 0 {
		sb.WriteString("allowed-tools:\n")
		for _, t := range tools {
			sb.WriteString(fmt.Sprintf("  - %s\n", t))
		}
	}
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", name))
	sb.WriteString("## When to use\n\n")
	sb.WriteString(fmt.Sprintf("Use this skill when %s.\n\n", strings.ToLower(description)))
	sb.WriteString("## Instructions\n\n")
	sb.WriteString("<!-- Add your skill instructions here -->\n")
	return sb.String()
}
