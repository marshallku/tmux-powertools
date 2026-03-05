package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/marshallku/tmux-powertools/internal/project"
	"github.com/marshallku/tmux-powertools/internal/tmux"
	"github.com/marshallku/tmux-powertools/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tmux-powertools",
	Short: "Project-aware tmux session manager",
	Long:  "Scan project directories, show git status, and manage tmux sessions with project-type layouts.",
	RunE:  runSelector,
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Kill all unattached tmux sessions",
	RunE:  runCleanup,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tmux sessions",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(listCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

func runSelector(cmd *cobra.Command, args []string) error {
	cfg := project.LoadConfig()
	projects := project.ScanProjects(cfg)

	if len(projects) == 0 {
		fmt.Println("No projects found. Configure roots in ~/.config/tmux-powertools/config.json")
		fmt.Println("Example: {\"roots\": [\"~/projects\", \"~/work\"]}")
		return nil
	}

	selected, err := ui.RunProjectSelector(projects)
	if err != nil {
		return err
	}

	if selected == nil {
		return nil
	}

	return ui.OpenProject(selected)
}

func runCleanup(cmd *cobra.Command, args []string) error {
	killed, err := tmux.CleanupSessions()
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	if len(killed) == 0 {
		fmt.Println("No unattached sessions to clean up.")
	} else {
		fmt.Printf("Killed %d session(s): %s\n", len(killed), strings.Join(killed, ", "))
	}
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	sessions, err := tmux.ListSessions()
	if err != nil {
		fmt.Println("No tmux server running.")
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions.")
		return nil
	}

	for _, s := range sessions {
		attached := ""
		if s.Attached {
			attached = " (attached)"
		}
		fmt.Printf("  %s — %d window(s)%s\n", s.Name, s.Windows, attached)
	}
	return nil
}

func init() {
	// Ensure config directory exists
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	os.MkdirAll(home+"/.config/tmux-powertools", 0755)
}
