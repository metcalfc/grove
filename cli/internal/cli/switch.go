package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/iheanyi/grove/internal/config"
	"github.com/iheanyi/grove/internal/registry"
	"github.com/iheanyi/grove/internal/worktree"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <worktree-name>",
	Short: "Open a new terminal tab/window in the specified worktree",
	Long: `Open a new terminal tab/window in the specified worktree.

On macOS, this uses osascript to open a new Terminal tab/window.
Optionally starts the dev server if not already running.

Examples:
  grove switch myrepo-feature-auth         # Switch to worktree
  grove switch myrepo-feature-auth --start # Switch and start dev server`,
	Args: cobra.ExactArgs(1),
	RunE: runSwitch,
}

func init() {
	switchCmd.Flags().Bool("start", false, "Start the dev server if not already running")
}

func runSwitch(cmd *cobra.Command, args []string) error {
	worktreeName := args[0]
	startServer, _ := cmd.Flags().GetBool("start")

	// Detect current worktree to find the main repo
	currentWt, err := worktree.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect current repository: %w", err)
	}

	// Determine the main repository path
	mainRepoPath := currentWt.Path
	if currentWt.IsWorktree && currentWt.MainWorktreePath != "" {
		mainRepoPath = currentWt.MainWorktreePath
	}

	// Find the target worktree
	worktreePath, err := findWorktree(mainRepoPath, worktreeName)
	if err != nil {
		return err
	}

	// Verify the worktree directory exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree directory does not exist: %s", worktreePath)
	}

	fmt.Printf("Switching to worktree: %s\n", worktreeName)
	fmt.Printf("Path: %s\n", worktreePath)

	// Open terminal based on platform
	if err := openTerminal(worktreePath); err != nil {
		return fmt.Errorf("failed to open terminal: %w", err)
	}

	// Optionally start the dev server
	if startServer {
		fmt.Println("\nStarting dev server...")

		// Load registry to check if already running
		reg, err := registry.Load()
		if err != nil {
			return fmt.Errorf("failed to load registry: %w", err)
		}

		// Detect the target worktree info
		targetWt, err := worktree.DetectAt(worktreePath)
		if err != nil {
			return fmt.Errorf("failed to detect target worktree: %w", err)
		}

		// Check if already running
		if existing, ok := reg.Get(targetWt.Name); ok && existing.IsRunning() {
			fmt.Printf("Server is already running at: %s\n", existing.URL)
		} else {
			fmt.Println("Note: Use 'grove start' in the new terminal to start the dev server")
			fmt.Println("(Auto-start from switch command would require backgrounding)")
		}
	}

	fmt.Println("\nTerminal opened successfully!")

	return nil
}

// findWorktree finds the path to a worktree given its name
func findWorktree(mainRepoPath, worktreeName string) (string, error) {
	// List all worktrees
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the worktree list
	lines := strings.Split(string(output), "\n")
	var currentPath string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			currentPath = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "branch ") && currentPath != "" {
			// Check if this worktree path matches the name
			baseName := filepath.Base(currentPath)
			if baseName == worktreeName {
				return currentPath, nil
			}
		}
	}

	// If not found by exact match, try parent directory + name
	parentDir := filepath.Dir(mainRepoPath)
	candidatePath := filepath.Join(parentDir, worktreeName)

	// Verify this is actually a worktree
	cmd = exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = mainRepoPath
	output, err = cmd.Output()
	if err == nil {
		if strings.Contains(string(output), candidatePath) {
			return candidatePath, nil
		}
	}

	return "", fmt.Errorf("worktree '%s' not found\nUse 'git worktree list' to see available worktrees", worktreeName)
}

// openTerminal opens a new terminal window/tab using the configured terminal
func openTerminal(path string) error {
	cfg, _ := config.Load("")
	terminal := cfg.Terminal

	// Auto-detect if not configured
	if terminal == "" {
		terminal = detectTerminal()
	}

	switch terminal {
	case "ghostty":
		return openGhostty(path)
	case "iterm":
		return openITerm(path)
	case "warp":
		return openWarp(path)
	case "terminal":
		return openAppleTerminal(path)
	default:
		if runtime.GOOS == "linux" {
			return openLinuxTerminal(path)
		}
		return openAppleTerminal(path)
	}
}

// detectTerminal returns the best available terminal on the current platform
func detectTerminal() string {
	if runtime.GOOS == "darwin" {
		if _, err := os.Stat("/Applications/Ghostty.app"); err == nil {
			return "ghostty"
		}
		return "terminal"
	}
	return "linux"
}

func openGhostty(path string) error {
	// Try CLI first
	ghosttyPaths := []string{
		"/Applications/Ghostty.app/Contents/MacOS/ghostty",
		"/opt/homebrew/bin/ghostty",
		filepath.Join(os.Getenv("HOME"), ".local/bin/ghostty"),
	}

	for _, p := range ghosttyPaths {
		if _, err := os.Stat(p); err == nil {
			cmd := exec.Command(p, "--working-directory="+path)
			return cmd.Start()
		}
	}

	// Fallback: open via AppleScript
	script := fmt.Sprintf(`
		tell application "Ghostty"
			activate
		end tell
		delay 0.5
		tell application "System Events"
			keystroke "cd %s" & return
		end tell
	`, shellEscape(path))

	return exec.Command("osascript", "-e", script).Run()
}

func openITerm(path string) error {
	script := fmt.Sprintf(`
		tell application "iTerm"
			activate
			try
				set newWindow to (create window with default profile)
				tell current session of newWindow
					write text "cd %s; clear"
				end tell
			on error
				tell current window
					create tab with default profile
					tell current session
						write text "cd %s; clear"
					end tell
				end tell
			end try
		end tell
	`, shellEscape(path), shellEscape(path))

	return exec.Command("osascript", "-e", script).Run()
}

func openWarp(path string) error {
	cmd := exec.Command("open", "-a", "Warp", path)
	return cmd.Run()
}

func openAppleTerminal(path string) error {
	script := fmt.Sprintf(`
		tell application "Terminal"
			activate
			tell application "System Events" to keystroke "t" using command down
			delay 0.5
			do script "cd %s; clear" in front window
		end tell
	`, shellEscape(path))

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		script = fmt.Sprintf(`
			tell application "Terminal"
				activate
				do script "cd %s; clear"
			end tell
		`, shellEscape(path))
		return exec.Command("osascript", "-e", script).Run()
	}
	return nil
}

func openLinuxTerminal(path string) error {
	terminals := []struct {
		name string
		args []string
	}{
		{"ghostty", []string{"--working-directory=" + path}},
		{"gnome-terminal", []string{"--working-directory=" + path}},
		{"konsole", []string{"--workdir", path}},
		{"xfce4-terminal", []string{"--working-directory=" + path}},
		{"xterm", []string{"-e", fmt.Sprintf("cd %s && $SHELL", shellEscape(path))}},
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term.name); err == nil {
			cmd := exec.Command(term.name, term.args...)
			if err := cmd.Start(); err == nil {
				return nil
			}
		}
	}

	return fmt.Errorf("no supported terminal emulator found\nPlease open the terminal manually at: %s", path)
}

// shellEscape escapes a string for safe use in shell commands
func shellEscape(s string) string {
	// Simple escape: wrap in single quotes and escape any single quotes
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}
