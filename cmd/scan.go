package cmd

import (
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

var scanCmd = &cobra.Command{
	Use:   "scan <path>",
	Short: "Deep security analysis of a skill",
	Long: `Deep security analysis of a skill using the full pattern database.

Scan performs thorough analysis including:
  - Prompt injection detection across all files
  - Script analysis for dangerous shell patterns
  - Quality checks (missing allowed-tools, weak descriptions)
  - Risk scoring with severity-weighted findings

Use 'coach lint' for quick spec validation during development.

Examples:
  coach scan ./my-skill           Full security scan
  coach scan ./my-skill --json    Output results as JSON`,
	Args:  cobra.ExactArgs(1),
	RunE:  runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	path := args[0]

	s, err := skill.Parse(path)
	if err != nil {
		return fmt.Errorf("parsing skill at %s: %w", path, err)
	}

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

	// Also scan all files in the skill directory for injection patterns
	fileFindings := scanner.ScanSkillFiles(s, db.Patterns)
	existingKeys := make(map[string]bool)
	for _, f := range result.Findings {
		key := fmt.Sprintf("%s:%s:%d", f.ID, f.File, f.Line)
		existingKeys[key] = true
	}
	for _, f := range fileFindings {
		key := fmt.Sprintf("%s:%s:%d", f.ID, f.File, f.Line)
		if !existingKeys[key] {
			result.Findings = append(result.Findings, f)
		}
	}

	result.Score = scanner.CalculateScore(result.Findings)
	result.Risk = pkg.RiskLevelFromScore(result.Score)

	fmt.Println()
	fmt.Println(ui.HeadingStyle.Render("  Security Scan Report"))
	fmt.Println()
	fmt.Println(ui.RenderScanSummary(result))
	fmt.Println()

	if len(result.Findings) > 0 {
		fmt.Println(ui.HeadingStyle.Render("  Findings"))
		fmt.Println()
		fmt.Println(ui.RenderFindings(result.Findings))
	}

	switch result.Risk {
	case pkg.RiskLow:
		fmt.Println(ui.SuccessStyle.Render("  Safe to install."))
	case pkg.RiskMedium:
		fmt.Println(ui.WarningStyle.Render("  Review warnings before installing."))
	case pkg.RiskHigh:
		fmt.Println(ui.ErrorStyle.Render("  Manual review recommended before installing."))
	case pkg.RiskCritical:
		fmt.Println(ui.ErrorStyle.Render("  DO NOT install without thorough review."))
		os.Exit(1)
	}
	fmt.Println()

	return nil
}
