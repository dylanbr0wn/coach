package pipeline

import (
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/scanner"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/types"
)

// Evaluate runs lint, scan, and quality checks on each candidate.
// The onProgress callback is called after each candidate is processed;
// it may be nil.
func Evaluate(candidates []SkillCandidate, db *types.PatternDatabase, force bool, onProgress func(current, total int, name string)) ([]VettedSkill, error) {
	vetted := make([]VettedSkill, 0, len(candidates))

	for i, c := range candidates {
		v := VettedSkill{
			Candidate:  c,
			Selectable: true,
		}
		name := filepath.Base(c.Path)

		// Lint: parse + validate.
		parsed, parseErr := skill.Parse(c.Path)
		if parseErr != nil {
			v.LintResult = CheckResult{Status: CheckFail, Issues: []string{parseErr.Error()}}
			v.Selectable = false
			vetted = append(vetted, v)
			callProgress(onProgress, i+1, len(candidates), name)
			continue
		}

		violations := skill.Validate(parsed)
		if len(violations) > 0 {
			v.Skill = parsed
			v.LintResult = CheckResult{Status: CheckFail, Issues: violations}
			v.Selectable = false
			vetted = append(vetted, v)
			callProgress(onProgress, i+1, len(candidates), name)
			continue
		}

		v.Skill = parsed
		v.LintResult = CheckResult{Status: CheckPass}

		// Scan: security analysis.
		scanResult, scanErr := scanner.ScanSkill(parsed, db)
		if scanErr != nil {
			v.ScanResult = nil
			v.LintResult = CheckResult{Status: CheckFail, Issues: []string{"scan error: " + scanErr.Error()}}
			v.Selectable = false
			vetted = append(vetted, v)
			callProgress(onProgress, i+1, len(candidates), name)
			continue
		}
		v.ScanResult = scanResult
		if scanResult.Risk == types.RiskCritical && !force {
			v.Selectable = false
		}

		// Quality: install-time heuristics.
		v.QualityResult = CheckQuality(parsed)

		vetted = append(vetted, v)
		callProgress(onProgress, i+1, len(candidates), name)
	}

	return vetted, nil
}

func callProgress(fn func(int, int, string), current, total int, name string) {
	if fn != nil {
		fn(current, total, name)
	}
}
