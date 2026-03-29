package rules

import _ "embed"

//go:embed patterns.yaml
var embeddedPatterns []byte

//go:embed agents.yaml
var embeddedAgents []byte
