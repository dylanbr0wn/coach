package ui

import (
	"fmt"
	"unicode/utf8"

	"github.com/charmbracelet/huh"
)

// SkillOption represents a skill choice in the interactive picker.
type SkillOption struct {
	Name        string
	Description string
	Dir         string
	Scope       string // "global" or "local"
}

// PickSkill presents an interactive huh.Select list of skills and returns the
// selected option's index, or -1 if the user cancelled. If createNew is true,
// a "Create new skill" option is prepended (returned as index -2).
func PickSkill(title string, skills []SkillOption, createNew bool) (int, error) {
	if len(skills) == 0 && !createNew {
		return -1, fmt.Errorf("no skills available")
	}

	type choice struct {
		index int
	}

	var options []huh.Option[int]

	if createNew {
		label := SuccessStyle.Render("+ Create new skill")
		options = append(options, huh.NewOption(label, -2))
	}

	for i, s := range skills {
		desc := s.Description
		if utf8.RuneCountInString(desc) > 50 {
			desc = string([]rune(desc)[:47]) + "..."
		}
		label := fmt.Sprintf("%s  %s  %s",
			InfoStyle.Render(s.Name),
			DimStyle.Render(desc),
			DimStyle.Render("("+s.Scope+")"),
		)
		options = append(options, huh.NewOption(label, i))
	}

	var selected int
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title(title).
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return -1, err
	}

	return selected, nil
}
