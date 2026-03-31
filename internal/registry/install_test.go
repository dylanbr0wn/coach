package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallSkill_Symlink(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: test\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentDir := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := InstallSkill(srcDir, agentDir, InstallOptions{})
	if err != nil {
		t.Fatalf("InstallSkill() error: %v", err)
	}

	destDir := filepath.Join(agentDir, filepath.Base(srcDir))
	info, err := os.Lstat(destDir)
	if err != nil {
		t.Fatalf("Lstat(%q) error: %v", destDir, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected symlink at %q, got mode %v", destDir, info.Mode())
	}
}

func TestInstallSkill_Copy(t *testing.T) {
	srcDir := t.TempDir()
	content := []byte("---\nname: test\n---\nbody")
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	agentDir := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := InstallSkill(srcDir, agentDir, InstallOptions{Copy: true})
	if err != nil {
		t.Fatalf("InstallSkill(Copy) error: %v", err)
	}

	destFile := filepath.Join(agentDir, filepath.Base(srcDir), "SKILL.md")
	got, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("ReadFile(%q) error: %v", destFile, err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}

	// Verify it's NOT a symlink.
	info, err := os.Lstat(filepath.Join(agentDir, filepath.Base(srcDir)))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("expected regular directory, got symlink")
	}
}

func TestInstallSkill_Overwrite(t *testing.T) {
	srcDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("v2"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentDir := filepath.Join(t.TempDir(), "skills")
	destDir := filepath.Join(agentDir, filepath.Base(srcDir))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := InstallSkill(srcDir, agentDir, InstallOptions{Copy: true})
	if err != nil {
		t.Fatalf("InstallSkill(overwrite) error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destDir, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v2" {
		t.Errorf("overwritten content = %q, want %q", got, "v2")
	}
}

func TestRecordInstall(t *testing.T) {
	coachDir := t.TempDir()

	err := RecordInstall(coachDir, "test-skill", "owner/repo", "abc123", 25, []string{"claude-code"})
	if err != nil {
		t.Fatalf("RecordInstall() error: %v", err)
	}

	provenance, err := LoadProvenance(coachDir)
	if err != nil {
		t.Fatalf("LoadProvenance() error: %v", err)
	}

	if len(provenance.Skills) != 1 {
		t.Fatalf("got %d skills, want 1", len(provenance.Skills))
	}

	s := provenance.Skills[0]
	if s.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", s.Name, "test-skill")
	}
	if s.Source != "owner/repo" {
		t.Errorf("Source = %q, want %q", s.Source, "owner/repo")
	}
	if s.CommitSHA != "abc123" {
		t.Errorf("CommitSHA = %q, want %q", s.CommitSHA, "abc123")
	}
	if s.RiskScore != 25 {
		t.Errorf("RiskScore = %d, want 25", s.RiskScore)
	}

	// Record again — should update, not duplicate.
	err = RecordInstall(coachDir, "test-skill", "owner/repo", "def456", 30, []string{"claude-code"})
	if err != nil {
		t.Fatalf("RecordInstall(update) error: %v", err)
	}

	provenance, err = LoadProvenance(coachDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(provenance.Skills) != 1 {
		t.Fatalf("got %d skills after update, want 1", len(provenance.Skills))
	}
	if provenance.Skills[0].CommitSHA != "def456" {
		t.Errorf("updated CommitSHA = %q, want %q", provenance.Skills[0].CommitSHA, "def456")
	}
}
