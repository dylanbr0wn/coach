package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dylanbr0wn/coach/pkg"
)

type InstallOptions struct {
	Copy        bool
	TargetAgent string
	Force       bool
}

func InstallSkill(srcDir string, agentSkillDir string, opts InstallOptions) error {
	skillName := filepath.Base(srcDir)
	destDir := filepath.Join(agentSkillDir, skillName)

	os.RemoveAll(destDir)

	if opts.Copy {
		return copyDir(srcDir, destDir)
	}

	absSrc, err := filepath.Abs(srcDir)
	if err != nil {
		return fmt.Errorf("resolving source path: %w", err)
	}

	if err := os.MkdirAll(agentSkillDir, 0755); err != nil {
		return fmt.Errorf("creating agent skill directory: %w", err)
	}

	return os.Symlink(absSrc, destDir)
}

func RecordInstall(coachDir string, name string, source string, sha string, score int, agents []string) error {
	provenance, err := LoadProvenance(coachDir)
	if err != nil {
		provenance = &InstalledSkills{}
	}

	provenance.AddSkill(pkg.InstalledSkill{
		Name:        name,
		Source:      source,
		CommitSHA:   sha,
		InstallDate: time.Now(),
		RiskScore:   score,
		Agents:      agents,
	})

	return SaveProvenance(coachDir, provenance)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
