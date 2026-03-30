package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylanbr0wn/coach/internal/ui"
	"github.com/spf13/cobra"
)

var version = "0.1.0-dev"

var rootCmd = &cobra.Command{
	Use:   "coach",
	Short: "Develop, test, and manage AI agent skills",
	Long:  "Coach is a CLI for authoring, testing, scanning, and managing AI agent skills across multiple coding agents.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = version
	rootCmd.SetHelpFunc(customHelp)
}

func customHelp(cmd *cobra.Command, args []string) {
	// Only customize help for the root command; subcommands get default help.
	if cmd != rootCmd {
		if cmd.Long != "" {
			fmt.Fprintln(os.Stdout, cmd.Long)
			fmt.Fprintln(os.Stdout)
		}
		_ = cmd.Usage()
		return
	}

	h := ui.HeadingStyle.Render
	d := ui.DimStyle.Render

	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", h("Coach")+" — "+d("v"+version))
	fmt.Fprintf(&b, "%s\n", d("Develop, test, and manage AI agent skills across multiple coding agents."))

	fmt.Fprintf(&b, "\n%s\n", h("Authoring"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "init"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "generate"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "edit"))

	fmt.Fprintf(&b, "\n%s\n", h("Analysis"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "lint"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "scan"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "preview"))

	fmt.Fprintf(&b, "\n%s\n", h("Management"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "install"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "list"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "sync"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "config"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "status"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "update-rules"))

	fmt.Fprintf(&b, "\n%s\n", h("Getting Started"))
	fmt.Fprintf(&b, "  1. coach init skill                         %s\n", d("Create a new skill"))
	fmt.Fprintf(&b, "  2. coach edit <name>                        %s\n", d("Write the skill content"))
	fmt.Fprintf(&b, "     coach generate <name>                    %s\n", d("Or use AI to author it"))
	fmt.Fprintf(&b, "  3. coach lint <path>                        %s\n", d("Validate the skill"))
	fmt.Fprintf(&b, "  4. coach config set distribute-to claude    %s\n", d("Configure distribution"))
	fmt.Fprintf(&b, "  5. coach sync                               %s\n", d("Symlink skills to agents"))

	fmt.Fprintf(&b, "\n%s\n", h("Flags"))
	fmt.Fprintf(&b, "%s\n", cmd.LocalFlags().FlagUsages())

	fmt.Fprintf(&b, "%s\n", d("Use \"coach [command] --help\" for more information about a command."))

	fmt.Fprint(os.Stdout, b.String())
}

func commandEntry(root *cobra.Command, name string) string {
	short := ""
	for _, c := range root.Commands() {
		if c.Name() == name {
			short = c.Short
			break
		}
	}
	padded := fmt.Sprintf("%-16s", name)
	return ui.InfoStyle.Render(padded) + short
}
