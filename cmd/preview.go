package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var previewCmd = &cobra.Command{
	Use:   "preview <path>",
	Short: "Render a SKILL.md in the terminal",
	Long:  "Shows parsed frontmatter, rendered markdown body, and file tree — exactly what an agent would see when the skill activates.",
	Args:  cobra.ExactArgs(1),
	RunE:  runPreview,
}

func init() {
	rootCmd.AddCommand(previewCmd)
}

func runPreview(cmd *cobra.Command, args []string) error {
	path := args[0]

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

func renderFileTree(dir string, indent string) {
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
