package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
)

var previewCmd = &cobra.Command{
	Use:   "preview [path]",
	Short: "Render a SKILL.md in the terminal",
	Long: `Shows parsed frontmatter, rendered markdown body, and file tree — exactly
what an agent would see when the skill activates.

If no path is given, an interactive picker lists all managed skills.

See also: coach edit (open in editor), coach lint (validation)`,
	Example: `  coach preview                    # Pick from managed skills interactively
  coach preview ./my-skill         # Preview a specific skill directory
  coach preview ~/.coach/skills/x  # Preview a global skill by path`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPreview,
}

func init() {
	rootCmd.AddCommand(previewCmd)
}

func runPreview(cmd *cobra.Command, args []string) error {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		coachDir := config.DefaultCoachDir()
		workDir, wdErr := os.Getwd()
		if wdErr != nil {
			return fmt.Errorf("getting working directory: %w", wdErr)
		}
		r := resolve.Resolver{
			GlobalSkillsDir: filepath.Join(coachDir, "skills"),
			WorkDir:         workDir,
		}
		name, pickErr := pickManagedSkill(&r, resolve.ScopeAny, "Select a skill to preview")
		if pickErr != nil {
			return pickErr
		}
		result, err := r.Resolve(name, resolve.ScopeAny)
		if err != nil {
			return err
		}
		path = result.Dir
	}

	s, err := skill.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing skill at %s: %w", path, err)
	}

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Skill Metadata"))
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.LabelStyle.Render("Name:"), s.Name)
	fmt.Printf("  %s %s\n", ui.LabelStyle.Render("Description:"), s.Description)
	if s.License != "" {
		fmt.Printf("  %s %s\n", ui.LabelStyle.Render("License:"), s.License)
	}
	if len(s.AllowedTools) > 0 {
		fmt.Printf("  %s %s\n", ui.LabelStyle.Render("Allowed tools:"), strings.Join(s.AllowedTools, ", "))
	}

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  File Tree"))
	fmt.Println()
	renderFileTree(path, "  ")

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Skill Body"))
	fmt.Println()

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		fmt.Println(s.Body)
		return nil
	}

	rendered, err := renderer.Render(s.Body)
	if err != nil {
		fmt.Println(s.Body)
		return nil
	}
	fmt.Print(rendered)

	return nil
}

func renderFileTree(dir, indent string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for i, entry := range entries {
		connector := "├── "
		if i == len(entries)-1 {
			connector = "└── "
		}

		name := entry.Name()
		if entry.IsDir() {
			fmt.Printf("%s%s%s/\n", indent, connector, name)
			childIndent := indent + "│   "
			if i == len(entries)-1 {
				childIndent = indent + "    "
			}
			renderFileTree(filepath.Join(dir, name), childIndent)
		} else {
			fmt.Printf("%s%s%s\n", indent, connector, name)
		}
	}
}
