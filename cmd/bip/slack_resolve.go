package main

import (
	"fmt"
	"io"
	"os"

	"github.com/matsen/bipartite/internal/config"
	"github.com/matsen/bipartite/internal/flow"
	"github.com/spf13/cobra"
)

var slackResolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Substitute Slack channel-mention markup with channel names",
	Long: `Read text on stdin, write text on stdout, replacing Slack channel-mention
markup (<#CXXXXXXXX> and <#CXXXXXXXX|alias>) with #channel-name.

Names come from sources.yml: the slack.channels block (reversed id->name) and
the slack.project_channels block (id->name). When an ID appears in both,
project_channels wins. Unknown IDs with no usable alias pass through unchanged.

This is a dumb text filter that never touches the Slack API. It makes no
assumptions about Markdown structure, so mentions inside code fences are
resolved too.

Examples:
  echo '• <#C03T8U5RATY>: notes' | bip slack resolve
  bip slack history fortnight-feats --since 2026-05-13 | bip slack resolve`,
	Args: cobra.NoArgs,
	RunE: runSlackResolve,
}

func init() {
	slackCmd.AddCommand(slackResolveCmd)
}

func runSlackResolve(cmd *cobra.Command, args []string) error {
	nexusPath := config.MustGetNexusPath()
	idToName, err := flow.LoadChannelIDMap(nexusPath)
	if err != nil {
		return outputSlackError(1, "config_error", err.Error())
	}

	// Read all of stdin. Mention markup never spans lines and fortnight-scale
	// input is well under a MB, so io.ReadAll is correct here; bufio.Scanner's
	// default 64 KB token limit would silently truncate larger input.
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return outputSlackError(1, "io_error", err.Error())
	}

	resolved := flow.ResolveChannelMentions(string(input), idToName)
	if _, err := fmt.Fprint(os.Stdout, resolved); err != nil {
		return outputSlackError(1, "io_error", err.Error())
	}
	return nil
}
