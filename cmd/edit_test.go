package cmd

import (
	"os"
	"testing"
)

func TestGetEditor(t *testing.T) {
	origEditor := os.Getenv("EDITOR")
	origVisual := os.Getenv("VISUAL")
	defer func() {
		os.Setenv("EDITOR", origEditor)
		os.Setenv("VISUAL", origVisual)
	}()

	// $EDITOR takes priority.
	os.Setenv("EDITOR", "nvim")
	os.Setenv("VISUAL", "code")
	editor, err := getEditor()
	if err != nil {
		t.Fatalf("getEditor failed: %v", err)
	}
	if editor != "nvim" {
		t.Errorf("editor = %q, want %q", editor, "nvim")
	}

	// Falls back to $VISUAL.
	os.Setenv("EDITOR", "")
	editor, err = getEditor()
	if err != nil {
		t.Fatalf("getEditor failed: %v", err)
	}
	if editor != "code" {
		t.Errorf("editor = %q, want %q", editor, "code")
	}
}

func TestFileHashAndChanged(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "skill-*.md")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("---\nname: test\n---\nbody")
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	hash, err := fileHash(f.Name())
	if err != nil {
		t.Fatalf("fileHash failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	// Same content → not changed.
	changed, err := fileChanged(f.Name(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("file should not be changed")
	}

	// Modify file → changed.
	if err := os.WriteFile(f.Name(), []byte("---\nname: test\n---\nupdated body"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err = fileChanged(f.Name(), hash)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("file should be changed after modification")
	}
}

func TestFileHashMissingFile(t *testing.T) {
	_, err := fileHash("/nonexistent/path/to/file.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
