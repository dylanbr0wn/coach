package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dylan/coach/internal/ui"
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
		cmd.Usage()
		return
	}

	h := ui.HeadingStyle.Render
	d := ui.DimStyle.Render

	var b strings.Builder

	fmt.Fprintf(&b, "%s\n", h("Coach")+" — "+d("v"+version))
	fmt.Fprintf(&b, "%s\n", d("Develop, test, and manage AI agent skills across multiple coding agents."))

	fmt.Fprintf(&b, "\n%s\n", h("Authoring"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "init"))

	fmt.Fprintf(&b, "\n%s\n", h("Analysis"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "lint"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "scan"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "preview"))

	fmt.Fprintf(&b, "\n%s\n", h("Management"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "install"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "status"))
	fmt.Fprintf(&b, "  %s\n", commandEntry(cmd, "update-rules"))

	fmt.Fprintf(&b, "\n%s\n", h("Getting Started"))
	fmt.Fprintf(&b, "  coach init skill\n")
	fmt.Fprintf(&b, "  coach lint ./skills/\n")
	fmt.Fprintf(&b, "  coach install claude\n")

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
