package cmd

import (
	"os"
	"testing"
)

func TestGetEditor(t *testing.T) {
	tests := []struct {
		name    string
		editor  string
		visual  string
		want    string
		wantErr bool
	}{
		{
			name:   "EDITOR takes priority",
			editor: "nvim",
			visual: "code",
			want:   "nvim",
		},
		{
			name:   "falls back to VISUAL",
			editor: "",
			visual: "code",
			want:   "code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("VISUAL", tt.visual)

			got, err := getEditor()
			if (err != nil) != tt.wantErr {
				t.Fatalf("getEditor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("getEditor() = %q, want %q", got, tt.want)
			}
		})
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
