package scanner

import "github.com/dylanbr0wn/coach/internal/types"

func ScanSkill(s *types.Skill, db *types.PatternDatabase) (*types.ScanResult, error) {
	var allFindings []types.Finding
	allFindings = append(allFindings, CheckInjection(s, db.Patterns)...)

	scriptFindings, err := CheckScripts(s, db.Patterns)
	if err != nil {
		return nil, err
	}
	allFindings = append(allFindings, scriptFindings...)
	allFindings = append(allFindings, CheckQuality(s)...)

	score := CalculateScore(allFindings)
	return &types.ScanResult{
		SkillPath: s.Path,
		Findings:  allFindings,
		Score:     score,
		Risk:      types.RiskLevelFromScore(score),
	}, nil
}
