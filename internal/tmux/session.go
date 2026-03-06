package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Session struct {
	Name     string
	Windows  int
	Attached bool
}

func ListSessions() ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}:#{session_windows}:#{session_attached}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		var windows int
		fmt.Sscanf(parts[1], "%d", &windows)
		sessions = append(sessions, Session{
			Name:     parts[0],
			Windows:  windows,
			Attached: parts[2] == "1",
		})
	}
	return sessions, nil
}

func SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

func CreateSession(name, path string) error {
	args := []string{"new-session", "-d", "-s", name}
	if path != "" {
		args = append(args, "-c", path)
	}
	return exec.Command("tmux", args...).Run()
}

func SwitchSession(name string) error {
	// If we're inside tmux, switch client
	if os.Getenv("TMUX") != "" {
		cmd := exec.Command("tmux", "switch-client", "-t", name)
		return cmd.Run()
	}
	// Otherwise attach
	cmd := exec.Command("tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func KillSession(name string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", name)
	return cmd.Run()
}

func ApplyLayout(sessionName, projectType string) error {
	exec.Command("tmux", "rename-window", "-t", sessionName+":0", "editor").Run()
	return nil
}

func CleanupSessions() ([]string, error) {
	sessions, err := ListSessions()
	if err != nil {
		return nil, err
	}

	var killed []string
	for _, s := range sessions {
		if !s.Attached {
			if err := KillSession(s.Name); err == nil {
				killed = append(killed, s.Name)
			}
		}
	}
	return killed, nil
}
