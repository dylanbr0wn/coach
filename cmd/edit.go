package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
)

var (
	editGlobal bool
	editLocal  bool
)

var editCmd = &cobra.Command{
	Use:   "edit [skill-name]",
	Short: "Open a skill in $EDITOR with lint-on-close",
	Long: `Opens a skill file in your editor and validates it after you save and close.
If validation issues are found, you can re-open to fix them.

If no skill name is given, an interactive picker lists all managed skills.

See also: coach generate (AI-assisted authoring), coach lint (validation)`,
	Example: `  coach edit                        # Pick from managed skills interactively
  coach edit code-reviewer          # Open in $EDITOR, lint on save
  coach edit code-reviewer -g       # Edit the global version
  coach edit deploy-check -l        # Edit the local (project) version`,
	Args: cobra.MaximumNArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().BoolVarP(&editGlobal, "global", "g", false, "Edit from global skills")
	editCmd.Flags().BoolVarP(&editLocal, "local", "l", false, "Edit from local project skills")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	editor, err := getEditor()
	if err != nil {
		return fmt.Errorf("no editor found (%w): set $EDITOR or $VISUAL environment variable", err)
	}

	coachDir := config.DefaultCoachDir()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: filepath.Join(coachDir, "skills"),
		WorkDir:         workDir,
	}

	scope := resolve.ScopeAny
	if editGlobal {
		scope = resolve.ScopeGlobal
	} else if editLocal {
		scope = resolve.ScopeLocal
	}

	var name string
	if len(args) > 0 {
		name = args[0]
	} else {
		picked, pickErr := pickManagedSkill(&r, scope, "Select a skill to edit")
		if pickErr != nil {
			return pickErr
		}
		name = picked
	}

	result, err := r.Resolve(name, scope)
	if err != nil {
		return err
	}

	initialHash, err := fileHash(result.Path)
	if err != nil {
		return fmt.Errorf("hashing file: %w", err)
	}

	currentHash := initialHash
	for {
		if err := openEditor(editor, result.Path); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		changed, err := fileChanged(result.Path, currentHash)
		if err != nil {
			return fmt.Errorf("checking file changes: %w", err)
		}

		if !changed {
			fmt.Println("No changes detected.")
			return nil
		}

		// Update hash for next iteration.
		currentHash, err = fileHash(result.Path)
		if err != nil {
			return fmt.Errorf("hashing file: %w", err)
		}

		s, err := skill.Parse(result.Dir)
		if err != nil {
			fmt.Println(ui.Error(fmt.Sprintf("Parse error: %v", err), ""))
			if ui.PromptYesNo("Re-open to fix? [Y/n] ") {
				continue
			}
			return nil
		}

		issues := skill.Validate(s)
		if len(issues) > 0 {
			fmt.Println()
			for _, issue := range issues {
				fmt.Println(ui.Error(issue, ""))
			}
			fmt.Println()
			if ui.PromptYesNo("Re-open to fix? [Y/n] ") {
				continue
			}
			return nil
		}

		fmt.Fprintln(os.Stderr, ui.Success(fmt.Sprintf("%s saved and validated", name)))
		fmt.Fprintln(os.Stderr, ui.NextStep(fmt.Sprintf("lint %s", name), "validate before distributing"))
		return nil
	}
}

// getEditor returns the editor to use, checking $EDITOR, $VISUAL, then vi.
func getEditor() (string, error) {
	if e := os.Getenv("EDITOR"); e != "" {
		return e, nil
	}
	if e := os.Getenv("VISUAL"); e != "" {
		return e, nil
	}
	path, err := exec.LookPath("vi")
	if err != nil {
		return "", fmt.Errorf("vi not found in PATH")
	}
	return path, nil
}

// fileHash returns the sha256 hex digest of the file at path.
func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// fileChanged reports whether the file at path has a different hash than previousHash.
func fileChanged(path, previousHash string) (bool, error) {
	current, err := fileHash(path)
	if err != nil {
		return false, err
	}
	return current != previousHash, nil
}

// openEditor opens the file at path in the given editor, connecting stdin/stdout/stderr.
func openEditor(editor, path string) error {
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
