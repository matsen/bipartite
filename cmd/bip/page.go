package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/matsen/bipartite/internal/config"
	"github.com/spf13/cobra"
)

// ntfyBaseURL is the ntfy server `bip page` posts to; tests point it at a local server.
var ntfyBaseURL = "https://ntfy.sh"

var (
	pageFromFlag   string
	pageLinkFlag   string
	pageCancelFlag bool
)

var pageCmd = &cobra.Command{
	Use:   "page <ask> | --cancel [<why>]",
	Short: "Ring the user's phone: this session needs them",
	Long: `Send one ntfy push saying this session needs attention.

The push is a doorbell, not the message: its title is "<session> needs you",
its body is the one-line ask plus the tmux window to go to, and tapping it
opens --link. Ring once per wait; the user's next turn in this session is the
clear. If the question resolves before the user answers, send --cancel so a
second push says the page is no longer needed.

The topic comes from $BIP_NTFY_TOPIC or ntfy_topic in ~/.config/bip/config.yml.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if !pageCancelFlag && len(args) == 0 {
			return fmt.Errorf("an ask is required unless --cancel is set")
		}
		return nil
	},
	RunE: runPage,
}

func init() {
	pageCmd.Flags().StringVar(&pageFromFlag, "from", "", "This session's name, as ListAgents shows it (required)")
	pageCmd.Flags().StringVar(&pageLinkFlag, "link", "", "Issue, PR, or comment URL the push opens when tapped")
	pageCmd.Flags().BoolVar(&pageCancelFlag, "cancel", false, "Say an earlier page from this session is no longer needed")
	_ = pageCmd.MarkFlagRequired("from")
	rootCmd.AddCommand(pageCmd)
}

func runPage(cmd *cobra.Command, args []string) error {
	topic := config.GetNtfyTopic()
	if topic == "" {
		exitWithError(ExitConfigError, "no ntfy topic: set ntfy_topic in %s or $BIP_NTFY_TOPIC", config.GlobalConfigPath())
	}
	title, body := pageMessage(pageFromFlag, strings.Join(args, " "), pageCancelFlag, tmuxWindow())
	if err := postNtfy(topic, title, body, pageLinkFlag); err != nil {
		exitWithError(ExitError, "%v", err)
	}
	fmt.Println("paged")
	return nil
}

// pageMessage builds the push's title and body. A cancel names no tmux window,
// since there is nothing to go to.
func pageMessage(from, text string, cancel bool, tmux string) (title, body string) {
	if cancel {
		if text == "" {
			text = "resolved without you"
		}
		return from + ": page not needed", text
	}
	if tmux != "" {
		text += "\ntmux: " + tmux
	}
	return from + " needs you", text
}

// tmuxWindow returns "session:window" for the pane this process runs in, or "" outside tmux.
func tmuxWindow() string {
	pane := os.Getenv("TMUX_PANE")
	if pane == "" {
		return ""
	}
	out, err := exec.Command("tmux", "display-message", "-p", "-t", pane, "#S:#W").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func postNtfy(topic, title, body, link string) error {
	req, err := http.NewRequest(http.MethodPost, ntfyBaseURL+"/"+topic, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", "bell")
	if link != "" {
		req.Header.Set("Click", link)
	}
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("posting to ntfy: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ntfy returned %s", resp.Status)
	}
	return nil
}
