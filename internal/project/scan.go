package project

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Project struct {
	Name      string
	Path      string
	GitBranch string
	GitDirty  bool
	GitAhead  int
	GitBehind int
	Type      string // "go", "node", "rust", "python", "generic"
}

type Config struct {
	Roots []string `json:"roots"`
}

func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	return Config{
		Roots: []string{
			filepath.Join(home, "projects"),
			filepath.Join(home, "work"),
		},
	}
}

func LoadConfig() Config {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "tmux-powertools", "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return DefaultConfig()
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig()
	}

	// Expand ~ in paths
	for i, root := range cfg.Roots {
		if strings.HasPrefix(root, "~/") {
			cfg.Roots[i] = filepath.Join(home, root[2:])
		}
	}

	return cfg
}

func ScanProjects(cfg Config) []Project {
	var projects []Project
	seen := make(map[string]bool)

	for _, root := range cfg.Roots {
		root = os.ExpandEnv(root)
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			fullPath := filepath.Join(root, entry.Name())
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			// Check if it's a git repo
			if _, err := os.Stat(filepath.Join(fullPath, ".git")); err != nil {
				continue
			}

			p := Project{
				Name: entry.Name(),
				Path: fullPath,
				Type: detectProjectType(fullPath),
			}

			p.GitBranch, p.GitDirty, p.GitAhead, p.GitBehind = getGitInfo(fullPath)
			projects = append(projects, p)
		}
	}

	return projects
}

func detectProjectType(path string) string {
	checks := map[string]string{
		"go.mod":           "go",
		"package.json":     "node",
		"Cargo.toml":       "rust",
		"pyproject.toml":   "python",
		"requirements.txt": "python",
	}

	for file, typ := range checks {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return typ
		}
	}

	return "generic"
}

func getGitInfo(path string) (branch string, dirty bool, ahead, behind int) {
	// Get branch name
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "unknown", false, 0, 0
	}
	branch = strings.TrimSpace(string(out))

	// Check dirty
	cmd = exec.Command("git", "-C", path, "status", "--porcelain")
	out, err = cmd.Output()
	if err == nil {
		dirty = len(strings.TrimSpace(string(out))) > 0
	}

	// Check ahead/behind
	cmd = exec.Command("git", "-C", path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	out, err = cmd.Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) == 2 {
			fmt.Sscanf(parts[0], "%d", &ahead)
			fmt.Sscanf(parts[1], "%d", &behind)
		}
	}

	return
}
