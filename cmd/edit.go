package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var (
	editGlobal bool
	editLocal  bool
)

var editCmd = &cobra.Command{
	Use:   "edit <skill-name>",
	Short: "Open a skill in $EDITOR with lint-on-close",
	Long:  "Opens a skill file in your editor and validates it after you save and close. If validation issues are found, you can re-open to fix them.",
	Example: `  coach edit code-reviewer          # Open in $EDITOR, lint on save
  coach edit code-reviewer -g       # Edit the global version
  coach edit deploy-check -l        # Edit the local (project) version`,
	Args: cobra.ExactArgs(1),
	RunE: runEdit,
}

func init() {
	editCmd.Flags().BoolVarP(&editGlobal, "global", "g", false, "Edit from global skills")
	editCmd.Flags().BoolVarP(&editLocal, "local", "l", false, "Edit from local project skills")
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	editor, err := getEditor()
	if err != nil {
		return fmt.Errorf("no editor found: set $EDITOR or $VISUAL environment variable")
	}

	coachDir := config.DefaultCoachDir()
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	r := resolve.Resolver{
		GlobalSkillsDir: coachDir + "/skills",
		WorkDir:         workDir,
	}

	scope := resolve.ScopeAny
	if editGlobal {
		scope = resolve.ScopeGlobal
	} else if editLocal {
		scope = resolve.ScopeLocal
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
			fmt.Println(ui.ErrorStyle.Render(fmt.Sprintf("Parse error: %v", err)))
			if promptReopen() {
				continue
			}
			return nil
		}

		issues := skill.Validate(s)
		if len(issues) > 0 {
			fmt.Println()
			for _, issue := range issues {
				fmt.Println(ui.ErrorStyle.Render("  ✗ " + issue))
			}
			fmt.Println()
			if promptReopen() {
				continue
			}
			return nil
		}

		fmt.Printf("%s %s saved and validated.\n", ui.SuccessStyle.Render("✓"), name)
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

// promptReopen asks the user whether to re-open the editor.
func promptReopen() bool {
	fmt.Print("Re-open to fix? [Y/n] ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}

// openEditor opens the file at path in the given editor, connecting stdin/stdout/stderr.
func openEditor(editor, path string) error {
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
