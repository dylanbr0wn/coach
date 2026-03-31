package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/rules"
	"github.com/dylanbr0wn/coach/internal/scanner"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/dylanbr0wn/coach/pkg"
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Deep security analysis of a skill",
	Long: `Deep security analysis of a skill using the full pattern database.

Scan performs thorough analysis including:
  - Prompt injection detection across all files
  - Script analysis for dangerous shell patterns
  - Quality checks (missing allowed-tools, weak descriptions)
  - Risk scoring with severity-weighted findings

See also: coach lint (quick spec validation), coach install (fetch and vet third-party skills)`,
	Example: `  coach scan                       # Scan all managed skills
  coach scan ./my-skill            # Full security scan of a specific skill
  coach scan ./my-skill --json     # Output results as JSON`,
	Args: cobra.MaximumNArgs(1),
	RunE: runScan,
}

func init() {
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return scanSingleSkill(args[0])
	}
	return scanAllManaged()
}

func scanAllManaged() error {
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
		fmt.Println(ui.Warn("No managed skills found.", ""))
		return nil
	}

	for _, m := range managed {
		if err := scanSingleSkill(m.Dir); err != nil {
			fmt.Fprintln(os.Stderr, ui.Error(fmt.Sprintf("%s: %v", m.Name, err), ""))
		}
	}
	return nil
}

func scanSingleSkill(path string) error {
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
	fmt.Println(ui.HeadingStyle.Render("  Scan: deep security analysis (full pattern database)"))
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
		fmt.Println(ui.Success("Safe to install."))
	case pkg.RiskMedium:
		fmt.Println(ui.Warn("Review warnings before installing.", ""))
	case pkg.RiskHigh:
		fmt.Println(ui.Error("Manual review recommended before installing.", ""))
	case pkg.RiskCritical:
		fmt.Println(ui.Error("DO NOT install without thorough review.", ""))
		os.Exit(1)
	}
	fmt.Println()

	return nil
}
