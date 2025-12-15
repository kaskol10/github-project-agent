package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginAgent represents a plugin-based agent loaded from .md files
type PluginAgent struct {
	Name       string
	Type       string // "core" or "custom"
	Purpose    string
	Triggers   []Trigger
	Guidelines map[string]interface{}
	Actions    []string
	Config     map[string]interface{}
	PromptPath string
	RawContent string
	FilePath   string
}

// Trigger defines when an agent should run
type Trigger struct {
	Event     string   // e.g., "issues.opened", "pull_request.opened"
	Schedule  string   // Cron expression
	Condition string   // e.g., "labels.contains('needs-review')"
	Manual    bool     // Can be triggered manually
	Labels    []string // Required labels
}

// LoadPlugins loads all agent plugins from the specified directory
func LoadPlugins(basePath string) ([]*PluginAgent, error) {
	var agents []*PluginAgent

	// Load from core directory
	corePath := filepath.Join(basePath, "core")
	if coreAgents, err := loadAgentsFromDir(corePath, "core"); err == nil {
		agents = append(agents, coreAgents...)
	}

	// Load from custom directory
	customPath := filepath.Join(basePath, "custom")
	if customAgents, err := loadAgentsFromDir(customPath, "custom"); err == nil {
		agents = append(agents, customAgents...)
	}

	return agents, nil
}

// loadAgentsFromDir loads all .md files from a directory as agents
func loadAgentsFromDir(dirPath, agentType string) ([]*PluginAgent, error) {
	var agents []*PluginAgent

	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return agents, nil // Directory doesn't exist, return empty
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		agent, err := loadAgentFromFile(filePath, agentType)
		if err != nil {
			// Log error but continue loading other agents
			fmt.Printf("Warning: failed to load agent from %s: %v\n", filePath, err)
			continue
		}

		if agent != nil {
			agents = append(agents, agent)
		}
	}

	return agents, nil
}

// loadAgentFromFile loads a single agent from a markdown file
func loadAgentFromFile(filePath, agentType string) (*PluginAgent, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	agent := &PluginAgent{
		Type:       agentType,
		RawContent: string(content),
		FilePath:   filePath,
		Guidelines: make(map[string]interface{}),
		Config:     make(map[string]interface{}),
	}

	// Parse the markdown file
	lines := strings.Split(string(content), "\n")

	var currentSection string
	var yamlBlock strings.Builder
	inYamlBlock := false

	for i, line := range lines {
		// Extract agent name from first heading
		if strings.HasPrefix(line, "# Agent:") {
			agent.Name = strings.TrimSpace(strings.TrimPrefix(line, "# Agent:"))
			continue
		}

		// Extract purpose
		if strings.HasPrefix(line, "**Purpose**:") {
			agent.Purpose = strings.TrimSpace(strings.TrimPrefix(line, "**Purpose**:"))
			continue
		}

		// Extract type
		if strings.HasPrefix(line, "**Type**:") {
			agent.Type = strings.TrimSpace(strings.TrimPrefix(line, "**Type**:"))
			continue
		}

		// Parse sections
		if strings.HasPrefix(line, "## ") {
			currentSection = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
			continue
		}

		// Parse triggers
		if currentSection == "trigger" || currentSection == "triggers" {
			agent.Triggers = parseTriggers(lines, i)
		}

		// Parse actions - only parse once when we first encounter a list item
		if currentSection == "actions" && len(agent.Actions) == 0 {
			// Check if this line looks like a list item (numbered or bulleted)
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "##") {
				// Check if it's a list item
				isListItem := strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") ||
					(len(trimmed) >= 3 && trimmed[0] >= '0' && trimmed[0] <= '9')
				if isListItem {
					// Start parsing from this line
					agent.Actions = parseListItems(lines, i)
				}
			}
		}

		// Parse configuration YAML block
		if strings.TrimSpace(line) == "```yaml" {
			inYamlBlock = true
			yamlBlock.Reset()
			continue
		}

		if inYamlBlock {
			if strings.TrimSpace(line) == "```" {
				// Parse YAML
				if err := yaml.Unmarshal([]byte(yamlBlock.String()), &agent.Config); err == nil {
					// Successfully parsed
				}
				inYamlBlock = false
				yamlBlock.Reset()
			} else {
				yamlBlock.WriteString(line)
				yamlBlock.WriteString("\n")
			}
		}

		// Extract prompt path
		if strings.Contains(line, "path:") && strings.Contains(strings.ToLower(line), "prompt") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				agent.PromptPath = strings.TrimSpace(parts[1])
			}
		}
	}

	return agent, nil
}

// parseTriggers extracts trigger information from markdown
func parseTriggers(lines []string, startIdx int) []Trigger {
	var triggers []Trigger
	currentTrigger := Trigger{}

	for i := startIdx; i < len(lines) && i < startIdx+20; i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "##") {
			if currentTrigger.Event != "" || currentTrigger.Schedule != "" || currentTrigger.Manual {
				triggers = append(triggers, currentTrigger)
				currentTrigger = Trigger{}
			}
			if strings.HasPrefix(line, "##") {
				break
			}
			continue
		}

		if strings.HasPrefix(line, "- event:") {
			currentTrigger.Event = strings.TrimSpace(strings.TrimPrefix(line, "- event:"))
		} else if strings.HasPrefix(line, "- schedule:") {
			schedule := strings.TrimSpace(strings.TrimPrefix(line, "- schedule:"))
			// Remove quotes if present
			schedule = strings.Trim(schedule, "\"")
			currentTrigger.Schedule = schedule
		} else if strings.HasPrefix(line, "- condition:") {
			currentTrigger.Condition = strings.TrimSpace(strings.TrimPrefix(line, "- condition:"))
		} else if strings.HasPrefix(line, "- manual:") {
			manualStr := strings.TrimSpace(strings.TrimPrefix(line, "- manual:"))
			currentTrigger.Manual = strings.ToLower(manualStr) == "true"
		} else if strings.HasPrefix(line, "- labels:") {
			labelsStr := strings.TrimSpace(strings.TrimPrefix(line, "- labels:"))
			currentTrigger.Labels = parseStringList(labelsStr)
		}
	}

	if currentTrigger.Event != "" || currentTrigger.Schedule != "" || currentTrigger.Manual {
		triggers = append(triggers, currentTrigger)
	}

	return triggers
}

// parseListItems extracts list items from markdown
func parseListItems(lines []string, startIdx int) []string {
	var items []string

	for i := startIdx; i < len(lines) && i < startIdx+50; i++ {
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "##") {
			if strings.HasPrefix(line, "##") {
				break
			}
			continue
		}

		// Handle bullet points
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			item := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			item = strings.TrimSpace(item)
			if item != "" {
				items = append(items, item)
			}
		} else {
			// Handle numbered lists (1., 2., 3., 10., etc.)
			// Match pattern: "N. " where N is any number followed by ". "
			if len(line) >= 3 && line[0] >= '0' && line[0] <= '9' {
				// Find where the number ends (could be multi-digit like 10., 100., etc.)
				numEndIdx := 0
				for numEndIdx < len(line) && line[numEndIdx] >= '0' && line[numEndIdx] <= '9' {
					numEndIdx++
				}
				// Check if we have ". " after the number
				if numEndIdx+1 < len(line) && line[numEndIdx] == '.' && line[numEndIdx+1] == ' ' {
					item := strings.TrimSpace(line[numEndIdx+2:])
					if item != "" {
						items = append(items, item)
					}
				}
			}
		}
	}

	return items
}

// parseStringList parses a comma-separated or array-like string list
func parseStringList(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")

	parts := strings.Split(s, ",")
	var result []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, "\"'")
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

// MatchTrigger checks if an agent should run based on the given event
func (a *PluginAgent) MatchTrigger(event string, labels []string) bool {
	for _, trigger := range a.Triggers {
		// Check event match
		if trigger.Event != "" && trigger.Event == event {
			// Check label conditions if specified
			if len(trigger.Labels) > 0 {
				hasAllLabels := true
				for _, requiredLabel := range trigger.Labels {
					found := false
					for _, label := range labels {
						if label == requiredLabel {
							found = true
							break
						}
					}
					if !found {
						hasAllLabels = false
						break
					}
				}
				if !hasAllLabels {
					continue
				}
			}
			return true
		}

		// Check manual trigger
		if trigger.Manual && event == "manual" {
			return true
		}
	}
	return false
}

// HasSchedule checks if the agent has a scheduled trigger
func (a *PluginAgent) HasSchedule() bool {
	for _, trigger := range a.Triggers {
		if trigger.Schedule != "" {
			return true
		}
	}
	return false
}

// GetSchedule returns the cron schedule if available
func (a *PluginAgent) GetSchedule() string {
	for _, trigger := range a.Triggers {
		if trigger.Schedule != "" {
			return trigger.Schedule
		}
	}
	return ""
}
