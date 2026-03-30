package pkg

import "time"

// Severity levels for scan findings
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ScorePoints returns how many risk points this severity contributes.
func (s Severity) ScorePoints() int {
	switch s {
	case SeverityCritical:
		return 30
	case SeverityHigh:
		return 15
	case SeverityMedium:
		return 5
	case SeverityWarning:
		return 2
	default:
		return 0
	}
}

// RiskLevel describes the overall risk assessment.
type RiskLevel int

const (
	RiskLow      RiskLevel = iota // 0-25
	RiskMedium                    // 26-50
	RiskHigh                      // 51-75
	RiskCritical                  // 76-100
)

func RiskLevelFromScore(score int) RiskLevel {
	switch {
	case score <= 25:
		return RiskLow
	case score <= 50:
		return RiskMedium
	case score <= 75:
		return RiskHigh
	default:
		return RiskCritical
	}
}

func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	case RiskCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// Skill represents a parsed SKILL.md file.
type Skill struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	License      string   `yaml:"license,omitempty"`
	AllowedTools []string `yaml:"allowed-tools,omitempty"`

	// Parsed from filesystem, not from YAML
	Path     string // Directory containing SKILL.md
	Body     string // Markdown body below frontmatter
	HasTests bool   // tests/ directory exists
}

// Finding represents a single lint or scan finding.
type Finding struct {
	ID          string // e.g. "PI-001"
	Category    string // e.g. "prompt-injection"
	Severity    Severity
	Name        string
	Description string
	File        string // Which file the finding was in
	Line        int    // Line number, 0 if not applicable
	Match       string // The matched text
}

// ScanResult is the output of scanning a skill.
type ScanResult struct {
	SkillPath string
	Findings  []Finding
	Score     int
	Risk      RiskLevel
}

// AgentConfig describes a known coding agent and where it stores skills.
type AgentConfig struct {
	Name            string   `yaml:"name"` // Display name (e.g. "Claude Code")
	SkillDir        string   `yaml:"skill_dir"`
	ProjectSkillDir string   `yaml:"project_skill_dir,omitempty"`
	ConfigFiles     []string `yaml:"config_files"`
	Supports        struct {
		Skills bool `yaml:"skills"`
		Hooks  bool `yaml:"hooks"`
		MCP    bool `yaml:"mcp"`
	} `yaml:"supports"`
}

// DetectedAgent is an agent found on the local system.
type DetectedAgent struct {
	Key       string // Registry key, e.g. "claude-code"
	Config    AgentConfig
	Installed bool   // Whether the agent's directory exists
	SkillDir  string // Resolved absolute path to skill directory
}

// InstalledSkill records provenance for an installed skill.
type InstalledSkill struct {
	Name        string    `yaml:"name"`
	Source      string    `yaml:"source"` // e.g. "owner/repo" or local path
	CommitSHA   string    `yaml:"commit_sha"`
	InstallDate time.Time `yaml:"install_date"`
	RiskScore   int       `yaml:"risk_score"`
	Agents      []string  `yaml:"agents"` // Which agents it was installed to
}

// Pattern is a single security detection rule.
type Pattern struct {
	ID          string   `yaml:"id"`
	Category    string   `yaml:"category"`
	Severity    string   `yaml:"severity"` // "critical", "high", "medium", "low"
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Regex       string   `yaml:"regex"`
	FileTypes   []string `yaml:"file_types"`
}

// PatternDatabase is the top-level structure for patterns.yaml.
type PatternDatabase struct {
	Patterns []Pattern `yaml:"patterns"`
}

// AgentRegistry is the top-level structure for agents.yaml.
type AgentRegistry struct {
	Agents map[string]AgentConfig `yaml:"agents"`
}
