package registry

import (
	"testing"
)

func TestParseSource_GitHubShorthand(t *testing.T) {
	src, err := ParseSource("vercel-labs/agent-skills")
	if err != nil {
		t.Fatalf("ParseSource() error: %v", err)
	}
	if src.Type != SourceGitHub {
		t.Errorf("Type = %v, want GitHub", src.Type)
	}
	if src.Owner != "vercel-labs" {
		t.Errorf("Owner = %q, want %q", src.Owner, "vercel-labs")
	}
	if src.Repo != "agent-skills" {
		t.Errorf("Repo = %q, want %q", src.Repo, "agent-skills")
	}
}

func TestParseSource_FullURL(t *testing.T) {
	src, err := ParseSource("https://github.com/vercel-labs/agent-skills")
	if err != nil {
		t.Fatalf("ParseSource() error: %v", err)
	}
	if src.Type != SourceGitHub {
		t.Errorf("Type = %v, want GitHub", src.Type)
	}
}

func TestParseSource_LocalPath(t *testing.T) {
	src, err := ParseSource("./my-skills")
	if err != nil {
		t.Fatalf("ParseSource() error: %v", err)
	}
	if src.Type != SourceLocal {
		t.Errorf("Type = %v, want Local", src.Type)
	}
}

func TestParseSource_Invalid(t *testing.T) {
	_, err := ParseSource("")
	if err == nil {
		t.Error("expected error for empty source")
	}
}
