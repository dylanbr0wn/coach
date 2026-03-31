package scanner

import "github.com/dylanbr0wn/coach/internal/types"

func CalculateScore(findings []types.Finding) int {
	idCount := make(map[string]int)
	score := 0
	for _, f := range findings {
		idCount[f.ID]++
		if idCount[f.ID] > 2 {
			continue
		}
		score += f.Severity.ScorePoints()
	}
	if score > 100 {
		score = 100
	}
	return score
}
