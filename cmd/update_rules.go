package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/internal/config"
	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

var updateRulesCmd = &cobra.Command{
	Use:   "update-rules",
	Short: "Fetch latest security patterns and agent registry",
	Long: `Downloads updated security scanning patterns and agent detection registry
from the remote rules repository.

See also: coach scan (uses security patterns), coach lint (uses security patterns)`,
	Example: `  coach update-rules               # Fetch latest patterns from remote
  coach scan                       # Run scan with updated patterns`,
	RunE: runUpdateRules,
}

func init() {
	rootCmd.AddCommand(updateRulesCmd)
}

func runUpdateRules(cmd *cobra.Command, args []string) error {
	coachDir := config.DefaultCoachDir()
	if err := config.EnsureCoachDir(coachDir); err != nil {
		return err
	}

	cfg, err := config.Load(coachDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	rulesDir := filepath.Join(coachDir, "rules")
	repoDir := filepath.Join(rulesDir, "repo")

	var sha string
	var alreadyUpToDate bool

	if spinErr := ui.WithSpinner(fmt.Sprintf("Fetching rules from %s", cfg.RulesSource), func() error {
		var fetchErr error
		sha, alreadyUpToDate, fetchErr = fetchRules(repoDir, cfg.RulesSource)
		return fetchErr
	}); spinErr != nil {
		return fmt.Errorf("fetching rules: %w", spinErr)
	}

	if alreadyUpToDate {
		fmt.Println(ui.Success("Already up to date."))
	} else {
		fmt.Println(ui.Success(fmt.Sprintf("Updated to %s", sha)))
	}

	return copyRuleFiles(repoDir, rulesDir)
}

func fetchRules(repoDir, source string) (sha string, upToDate bool, err error) {
	if _, statErr := os.Stat(filepath.Join(repoDir, ".git")); statErr == nil {
		repo, openErr := git.PlainOpen(repoDir)
		if openErr == nil {
			w, wtErr := repo.Worktree()
			if wtErr == nil {
				pullErr := w.Pull(&git.PullOptions{Force: true})
				if pullErr != nil && pullErr != git.NoErrAlreadyUpToDate {
					os.RemoveAll(repoDir)
				} else {
					head, _ := repo.Head()
					sha = "unknown"
					if head != nil {
						sha = head.Hash().String()[:12]
					}
					return sha, pullErr == git.NoErrAlreadyUpToDate, nil
				}
			}
		}
	}

	os.RemoveAll(repoDir)
	repo, cloneErr := git.PlainClone(repoDir, false, &git.CloneOptions{
		URL:           source,
		Depth:         1,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
	})
	if cloneErr != nil {
		return "", false, fmt.Errorf("cloning rules repository: %w\n\n  If the rules repo doesn't exist yet, this is expected.\n  Coach will use its embedded patterns in the meantime.", cloneErr)
	}

	head, _ := repo.Head()
	sha = "unknown"
	if head != nil {
		sha = head.Hash().String()[:12]
	}
	return sha, false, nil
}

func copyRuleFiles(repoDir, rulesDir string) error {
	files := []string{"patterns.yaml", "agents.yaml"}
	copied := 0

	for _, f := range files {
		src := filepath.Join(repoDir, f)
		dst := filepath.Join(rulesDir, f)

		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}

		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", f, err)
		}
		copied++
	}

	if copied > 0 {
		fmt.Println(ui.Success(fmt.Sprintf("Updated %d rule files", copied)))
	}
	fmt.Println()

	return nil
}
