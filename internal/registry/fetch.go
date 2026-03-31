package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"

	"github.com/dylanbr0wn/coach/internal/config"
)

type SourceType int

const (
	SourceLocal SourceType = iota
	SourceGitHub
	SourceURL
)

type Source struct {
	Type     SourceType
	Raw      string
	Owner    string
	Repo     string
	CloneURL string
	Path     string
}

func ParseSource(input string) (*Source, error) {
	if input == "" {
		return nil, fmt.Errorf("source cannot be empty")
	}

	if strings.HasPrefix(input, "./") || strings.HasPrefix(input, "/") || strings.HasPrefix(input, "../") {
		return &Source{
			Type: SourceLocal,
			Raw:  input,
			Path: input,
		}, nil
	}

	if strings.HasPrefix(input, "https://github.com/") {
		trimmed := strings.TrimPrefix(input, "https://github.com/")
		trimmed = strings.TrimSuffix(trimmed, ".git")
		trimmed = strings.TrimSuffix(trimmed, "/")
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid GitHub URL: %s", input)
		}
		return &Source{
			Type:     SourceGitHub,
			Raw:      input,
			Owner:    parts[0],
			Repo:     parts[1],
			CloneURL: "https://github.com/" + parts[0] + "/" + parts[1] + ".git",
		}, nil
	}

	if strings.HasPrefix(input, "https://") || strings.HasPrefix(input, "git@") {
		return &Source{
			Type:     SourceURL,
			Raw:      input,
			CloneURL: input,
		}, nil
	}

	parts := strings.SplitN(input, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return &Source{
			Type:     SourceGitHub,
			Raw:      input,
			Owner:    parts[0],
			Repo:     parts[1],
			CloneURL: "https://github.com/" + parts[0] + "/" + parts[1] + ".git",
		}, nil
	}

	return nil, fmt.Errorf("cannot parse source: %s (use owner/repo, a URL, or a local path)", input)
}

func FetchToCache(src *Source) (localPath, commitSHA string, err error) {
	if src.Type == SourceLocal {
		absPath, err := filepath.Abs(src.Path)
		if err != nil {
			return "", "", err
		}
		return absPath, "local", nil
	}

	coachDir := config.DefaultCoachDir()
	cacheDir := filepath.Join(coachDir, "cache")
	if err := config.EnsureCoachDir(coachDir); err != nil {
		return "", "", fmt.Errorf("ensuring coach dir: %w", err)
	}

	repoDir := src.Owner + "_" + src.Repo
	if src.Type == SourceURL {
		repoDir = sanitizePath(src.CloneURL)
	}
	destPath := filepath.Join(cacheDir, repoDir)

	if _, statErr := os.Stat(destPath); statErr == nil {
		if path, sha, ok := tryUpdateCache(destPath); ok {
			return path, sha, nil
		}
		os.RemoveAll(destPath)
	}

	repo, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:           src.CloneURL,
		Depth:         1,
		ReferenceName: plumbing.HEAD,
		SingleBranch:  true,
	})
	if err != nil {
		return "", "", fmt.Errorf("cloning %s: %w", src.CloneURL, err)
	}

	head, err := repo.Head()
	sha := "unknown"
	if err == nil {
		sha = head.Hash().String()[:12]
	}

	return destPath, sha, nil
}

func FindSkills(dir string) ([]string, error) {
	var skills []string

	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
		skills = append(skills, dir)
		return skills, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			skillPath := filepath.Join(dir, e.Name(), "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				skills = append(skills, filepath.Join(dir, e.Name()))
			}
		}
	}

	return skills, nil
}

// tryUpdateCache attempts to pull the latest changes for an existing cached repo.
// Returns the path, short SHA, and true on success; false if the cache should be rebuilt.
func tryUpdateCache(destPath string) (string, string, bool) {
	repo, err := git.PlainOpen(destPath)
	if err != nil {
		return "", "", false
	}

	if w, err := repo.Worktree(); err == nil {
		_ = w.Pull(&git.PullOptions{Force: true})
	}

	head, err := repo.Head()
	if err != nil {
		return destPath, "unknown", true
	}
	return destPath, head.Hash().String()[:12], true
}

func sanitizePath(url string) string {
	r := strings.NewReplacer("/", "_", ":", "_", ".", "_")
	return r.Replace(url)
}
