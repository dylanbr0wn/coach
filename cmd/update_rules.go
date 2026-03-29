package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylan/coach/internal/config"
	"github.com/dylan/coach/internal/ui"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/spf13/cobra"
)

var updateRulesCmd = &cobra.Command{
	Use:   "update-rules",
	Short: "Fetch latest security patterns and agent registry",
	Long:  "Downloads updated security scanning patterns and agent detection registry from the remote rules repository.",
	RunE:  runUpdateRules,
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

	fmt.Printf("  Fetching rules from %s...\n", ui.DimStyle.Render(cfg.RulesSource))

	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err == nil {
		repo, err := git.PlainOpen(repoDir)
		if err == nil {
			w, err := repo.Worktree()
			if err == nil {
				err = w.Pull(&git.PullOptions{Force: true})
				if err != nil && err != git.NoErrAlreadyUpToDate {
					os.RemoveAll(repoDir)
				} else {
					head, _ := repo.Head()
					sha := "unknown"
					if head != nil {
						sha = head.Hash().String()[:12]
					}
					if err == git.NoErrAlreadyUpToDate {
						fmt.Printf("  %s\n", ui.SuccessStyle.Render("Already up to date."))
					} else {
						fmt.Printf("  %s Updated to %s\n", ui.SuccessStyle.Render("✓"), sha)
					}
					return copyRuleFiles(repoDir, rulesDir)
				}
			}
		}
	}

	os.RemoveAll(repoDir)
	repo, err := git.PlainClone(repoDir, false, &git.CloneOptions{
		URL:           cfg.RulesSource,
		Depth:         1,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
	})
	if err != nil {
		return fmt.Errorf("cloning rules repository: %w\n\n  If the rules repo doesn't exist yet, this is expected.\n  Coach will use its embedded patterns in the meantime.", err)
	}

	head, _ := repo.Head()
	sha := "unknown"
	if head != nil {
		sha = head.Hash().String()[:12]
	}
	fmt.Printf("  %s Fetched rules at %s\n", ui.SuccessStyle.Render("✓"), sha)

	return copyRuleFiles(repoDir, rulesDir)
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

		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", f, err)
		}
		copied++
	}

	if copied > 0 {
		fmt.Printf("  %s Updated %d rule files\n", ui.SuccessStyle.Render("✓"), copied)
	}
	fmt.Println()

	return nil
}
