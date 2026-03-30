package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dylanbr0wn/coach/pkg"
	"gopkg.in/yaml.v3"
)

type InstalledSkills struct {
	Skills []pkg.InstalledSkill `yaml:"skills"`
}

func LoadProvenance(coachDir string) (*InstalledSkills, error) {
	path := filepath.Join(coachDir, "installed.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &InstalledSkills{}, nil
		}
		return nil, fmt.Errorf("reading provenance: %w", err)
	}

	var installed InstalledSkills
	if err := yaml.Unmarshal(data, &installed); err != nil {
		return nil, fmt.Errorf("parsing provenance: %w", err)
	}
	return &installed, nil
}

func SaveProvenance(coachDir string, installed *InstalledSkills) error {
	path := filepath.Join(coachDir, "installed.yaml")
	data, err := yaml.Marshal(installed)
	if err != nil {
		return fmt.Errorf("marshaling provenance: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func (is *InstalledSkills) AddSkill(skill pkg.InstalledSkill) {
	for i, existing := range is.Skills {
		if existing.Name == skill.Name {
			is.Skills[i] = skill
			return
		}
	}
	is.Skills = append(is.Skills, skill)
}
