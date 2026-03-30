package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/dylanbr0wn/coach/internal/config"
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

var (
	initGlobal bool
	initLocal  bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.AddCommand(initSkillCmd)
	initSkillCmd.Flags().BoolVarP(&initGlobal, "global", "g", false, "Create in global skills (~/.coach/skills/)")
	initSkillCmd.Flags().BoolVarP(&initLocal, "local", "l", false, "Create in local project skills (.coach/skills/)")
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

	if strings.TrimSpace(description) == "" {
		return fmt.Errorf("description is required")
	}

	// Determine target directory
	var dir string

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	if initLocal {
		dir = filepath.Join(workDir, ".coach", "skills", name)
	} else if initGlobal {
		dir = filepath.Join(config.DefaultCoachDir(), "skills", name)
	} else {
		cfg, err := config.Load(config.DefaultCoachDir())
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.DefaultScope == "local" {
			dir = filepath.Join(workDir, ".coach", "skills", name)
		} else {
			dir = filepath.Join(config.DefaultCoachDir(), "skills", name)
		}
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
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
	if err := os.WriteFile(skillPath, []byte(skillContent), 0o644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	if includeTests {
		if err := os.MkdirAll(filepath.Join(dir, "tests"), 0o755); err != nil {
			return fmt.Errorf("creating tests directory: %w", err)
		}
	}
	if includeScripts {
		if err := os.MkdirAll(filepath.Join(dir, "scripts"), 0o755); err != nil {
			return fmt.Errorf("creating scripts directory: %w", err)
		}
	}
	if includeRefs {
		if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
			return fmt.Errorf("creating references directory: %w", err)
		}
	}

	fmt.Println()
	fmt.Printf("  %s Skill created: %s\n", ui.SuccessStyle.Render("✓"), name)
	fmt.Printf("  Path: %s\n", dir)
	fmt.Println()
	boldStyle := lipgloss.NewStyle().Bold(true)
	fmt.Printf("  Next steps:\n")
	fmt.Printf("    %-36s   Edit the skill\n", boldStyle.Render("coach edit "+name))
	fmt.Printf("    %-36s   Author with AI\n", boldStyle.Render("coach generate "+name))
	fmt.Printf("    %-36s   Validate all managed skills\n", boldStyle.Render("coach lint"))
	fmt.Printf("    %-36s   Distribute to your agents\n", boldStyle.Render("coach sync"))
	fmt.Println()

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
