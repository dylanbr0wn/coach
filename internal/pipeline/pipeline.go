// Package pipeline implements the batch-aware install pipeline:
// Discover → Evaluate → Present → Commit.
package pipeline

import "github.com/dylanbr0wn/coach/internal/types"

// OriginType describes where a skill candidate was found.
type OriginType int

const (
	// OriginLocal is a skill discovered from a local directory path.
	OriginLocal OriginType = iota
	// OriginRemote is a skill discovered from a remote GitHub source.
	OriginRemote
	// OriginInstalledUntracked is a skill found in an agent directory
	// that has no matching provenance record.
	OriginInstalledUntracked
	// OriginInstalledModified is a skill found in an agent directory
	// whose content differs from the provenance record's content hash.
	OriginInstalledModified
)

func (o OriginType) String() string {
	switch o {
	case OriginLocal:
		return "local"
	case OriginRemote:
		return "remote"
	case OriginInstalledUntracked:
		return "untracked"
	case OriginInstalledModified:
		return "modified"
	default:
		return "unknown"
	}
}

// SkillCandidate is a skill discovered by the Discover stage, before evaluation.
type SkillCandidate struct {
	Path   string     // absolute path to skill directory
	Source string     // original source string for provenance
	SHA    string     // commit SHA if remote, "local" otherwise
	Origin OriginType // how the candidate was found
}

// CheckStatus represents the outcome of a single evaluation check.
type CheckStatus int

const (
	// CheckPass means the check found no issues.
	CheckPass CheckStatus = iota
	// CheckWarn means the check found non-blocking warnings.
	CheckWarn
	// CheckFail means the check found blocking issues.
	CheckFail
)

func (s CheckStatus) String() string {
	switch s {
	case CheckPass:
		return "PASS"
	case CheckWarn:
		return "WARN"
	case CheckFail:
		return "FAIL"
	default:
		return "—"
	}
}

// CheckResult holds the outcome and issue details for a single evaluation check.
type CheckResult struct {
	Status CheckStatus
	Issues []string
}

// VettedSkill is the output of the Evaluate stage: a candidate with all checks run.
type VettedSkill struct {
	Candidate     SkillCandidate
	Skill         *types.Skill      // parsed skill; nil if lint failed
	LintResult    CheckResult       // spec compliance
	ScanResult    *types.ScanResult // security scan; nil if lint failed
	QualityResult CheckResult       // quality heuristics; skipped if lint failed
	Selectable    bool              // false if lint FAIL or scan CRITICAL (without --force)
}

// InstallOptions controls how Commit installs skills.
type InstallOptions struct {
	Copy   bool                  // copy instead of symlink
	Force  bool                  // allow CRITICAL skills
	Scope  string                // "global" or "local"
	Agents []types.DetectedAgent // target agents for distribution
}
