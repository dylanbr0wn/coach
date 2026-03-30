package scanner

import "github.com/dylanbr0wn/coach/pkg"

func ScanSkill(s *pkg.Skill, db *pkg.PatternDatabase) *pkg.ScanResult {
	var allFindings []pkg.Finding
	allFindings = append(allFindings, CheckInjection(s, db.Patterns)...)
	allFindings = append(allFindings, CheckScripts(s, db.Patterns)...)
	allFindings = append(allFindings, CheckQuality(s)...)

	score := CalculateScore(allFindings)
	return &pkg.ScanResult{
		SkillPath: s.Path,
		Findings:  allFindings,
		Score:     score,
		Risk:      pkg.RiskLevelFromScore(score),
	}
}
