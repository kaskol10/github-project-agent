package agent

import (
	"os"
	"path/filepath"
)

// getPromptPath finds the prompts directory
// This is a shared helper used by all agents
func getPromptPath(defaultPath string) string {
	// Try current directory first
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}
	
	// Try relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		candidate := filepath.Join(execDir, defaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	
	// Try working directory
	wd, err := os.Getwd()
	if err == nil {
		candidate := filepath.Join(wd, defaultPath)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	
	return defaultPath
}

