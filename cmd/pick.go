package cmd

import (
	"fmt"

	"github.com/dylanbr0wn/coach/internal/resolve"
	"github.com/dylanbr0wn/coach/internal/skill"
	"github.com/dylanbr0wn/coach/internal/ui"
)

// pickManagedSkill lists managed skills via the resolver and presents an
// interactive picker. Returns the selected skill name. If createNew is needed,
// use pickManagedSkillOrNew instead.
func pickManagedSkill(r *resolve.Resolver, scope resolve.Scope, title string) (string, error) {
	name, isNew, err := pickManagedSkillOrNew(r, scope, title, false)
	if err != nil {
		return "", err
	}
	if isNew {
		return "", fmt.Errorf("unexpected create-new selection")
	}
	return name, nil
}

// pickManagedSkillOrNew lists managed skills and optionally a "Create new skill"
// option. Returns (skillName, isNew, error).
func pickManagedSkillOrNew(r *resolve.Resolver, scope resolve.Scope, title string, showCreateNew bool) (string, bool, error) {
	managed, err := r.List(scope)
	if err != nil {
		return "", false, fmt.Errorf("listing managed skills: %w", err)
	}

	if len(managed) == 0 && !showCreateNew {
		fmt.Println()
		fmt.Printf("  %s\n", ui.DimStyle.Render("No managed skills found."))
		fmt.Printf("  Run %s to create one.\n\n", ui.InfoStyle.Render("coach init skill"))
		return "", false, fmt.Errorf("no managed skills found")
	}

	var options []ui.SkillOption
	for _, m := range managed {
		scopeLabel := "global"
		if m.Scope == resolve.ScopeLocal {
			scopeLabel = "local"
		}
		desc := ""
		parsed, parseErr := skill.Parse(m.Dir)
		if parseErr == nil {
			desc = parsed.Description
		}
		options = append(options, ui.SkillOption{
			Name:        m.Name,
			Description: desc,
			Dir:         m.Dir,
			Scope:       scopeLabel,
		})
	}

	idx, err := ui.PickSkill(title, options, showCreateNew)
	if err != nil {
		return "", false, err
	}

	if idx == ui.PickCreateNew {
		return "", true, nil
	}

	return managed[idx].Name, false, nil
}
