package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dylanbr0wn/coach/internal/config"
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
	Long:  "Validates SKILL.md frontmatter, checks for prompt injection patterns, and audits scripts for dangerous operations.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runLint,
}

func init() {
	lintCmd.Flags().BoolVar(&lintJSON, "json", false, "Output results as JSON")
	rootCmd.AddCommand(lintCmd)
}

func runLint(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

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
	fmt.Println(ui.RenderScanSummary(result))
	fmt.Println()
	fmt.Println(ui.RenderFindings(result.Findings))

	for _, f := range result.Findings {
		if f.Severity >= pkg.SeverityHigh {
			os.Exit(1)
		}
	}

	return nil
}

func outputJSON(result *pkg.ScanResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
