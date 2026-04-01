package pipeline

import (
	"strings"

	"github.com/dylanbr0wn/coach/internal/types"
)

// CheckQuality runs install-time quality heuristics on a parsed skill.
// All issues are warnings — they never block installation.
func CheckQuality(s *types.Skill) CheckResult {
	var issues []string

	if len(s.Description) < 20 {
		issues = append(issues, "description under 20 chars")
	}

	descLower := strings.ToLower(s.Description)
	if !strings.Contains(descLower, "use when") && !strings.Contains(descLower, "use this") {
		issues = append(issues, "no trigger phrase in description (\"Use when\" / \"Use this\")")
	}

	if len(s.AllowedTools) == 0 {
		issues = append(issues, "no allowed-tools declared")
	}

	bodyLower := strings.ToLower(s.Body)
	if !strings.Contains(bodyLower, "when to use") && !strings.Contains(bodyLower, "## when") {
		issues = append(issues, "no \"When to Use\" section in body")
	}

	if len(strings.TrimSpace(s.Body)) < 50 {
		issues = append(issues, "body under 50 chars (suspiciously thin)")
	}

	if len(issues) > 0 {
		return CheckResult{Status: CheckWarn, Issues: issues}
	}
	return CheckResult{Status: CheckPass}
}
