package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/scanner"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/dylanbr0wn/coach/pkg"
	"github.com/spf13/cobra"
)

var lintJSON bool

var lintCmd = &cobra.Command{
	Use:   "lint [path]",
	Short: "Check a skill for spec compliance and security issues",
	Long: `Validate a skill against the SKILL.md specification and run security checks.

Lint checks for:
  - Required frontmatter fields (name, description)
  - Field format and length constraints
  - Body content presence
  - Common security patterns (prompt injection, dangerous commands)

Use 'coach scan' for deeper security analysis with the full pattern database.

Examples:
  coach lint .                    Lint skill in current directory
  coach lint ./my-skill           Lint a specific skill
  coach lint ./my-skill --json    Output results as JSON`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLint,
}

func init() {
	lintCmd.Flags().BoolVar(&lintJSON, "json", false, "Output results as JSON")
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return lintSingleSkill(args[0])
	}
	return lintAllManaged()
}

func lintAllManaged() error {
	coachDir := config.DefaultCoachDir()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         workDir,
	}

	managed, err := r.List(resolve.ScopeAny)
	if err != nil {
		return fmt.Errorf("listing managed skills: %w", err)
	}

	if len(managed) == 0 {
		fmt.Println("No managed skills found.")
		return nil
	}

	var exitCode int
	for _, m := range managed {
		if err := lintSingleSkill(m.Dir); err != nil {
			exitCode = 1
		}
	}

	if exitCode != 0 {
		os.Exit(1)
	}
	return nil
}

func lintSingleSkill(path string) error {
	s, err := skill.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing skill at %s: %w", path, err)
	}

	validationErrors := skill.Validate(s)

	coachDir := config.DefaultCoachDir()
	rulesDir := ""
	if coachDir != "" {
		rulesDir = coachDir + "/rules"
	}
	db, err := rules.LoadPatterns(rulesDir)
	if err != nil {
		return fmt.Errorf("loading patterns: %w", err)
	}

	result := scanner.ScanSkill(s, db)

	for _, ve := range validationErrors {
		result.Findings = append([]pkg.Finding{{
			ID:          "SPEC-001",
			Category:    "spec-compliance",
			Severity:    pkg.SeverityHigh,
			Name:        "Spec violation",
			Description: ve,
			File:        s.Path + "/SKILL.md",
		}}, result.Findings...)
		result.Score = scanner.CalculateScore(result.Findings)
		result.Risk = pkg.RiskLevelFromScore(result.Score)
	}

	if lintJSON {
		return outputJSON(result)
	}

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Lint: spec compliance + basic security"))
	fmt.Println()
	fmt.Println(ui.RenderScanSummary(result))
	fmt.Println()
	fmt.Println(ui.RenderFindings(result.Findings))

	counts := countFindingSeverities(result.Findings)
	fmt.Printf("  %s  %d error(s), %d warning(s) %s\n\n",
		ui.HeadingStyle.Render("Lint complete."),
		counts["errors"], counts["warnings"],
		ui.DimStyle.Render("(spec + basic security)"),
	)

	for _, f := range result.Findings {
		if f.Severity >= pkg.SeverityHigh {
			return fmt.Errorf("high severity finding: %s", f.Name)
		}
	}

	return nil
}

func countFindingSeverities(findings []pkg.Finding) map[string]int {
	counts := map[string]int{"errors": 0, "warnings": 0}
	for _, f := range findings {
		if f.Severity >= pkg.SeverityHigh {
			counts["errors"]++
		} else if f.Severity >= pkg.SeverityWarning {
			counts["warnings"]++
		}
	}
	return counts
}

func outputJSON(result *pkg.ScanResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
