package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kaskol10/github-project-agent/config"
	"github.com/kaskol10/github-project-agent/github"
	"github.com/kaskol10/github-project-agent/llm"
	"github.com/kaskol10/github-project-agent/plugins"
	"github.com/kaskol10/github-project-agent/prompts"
)

// MCPInterface provides a Model Context Protocol compatible interface
// for agent interactions with GitHub
type MCPInterface struct {
	githubClient   github.UnifiedClient
	pluginAgents   []*plugins.PluginAgent
	pluginExecutor *plugins.PluginExecutor
	llmClient      interface{} // *llm.Client - using interface{} to avoid circular import
	guidelines     interface{} // *guidelines.Guidelines - using interface{} to avoid circular import
	config         interface{} // *config.Config - for accessing task format rules
}

// NewMCPInterface creates a new MCP-compatible interface
func NewMCPInterface(ghClient github.UnifiedClient, pluginAgents []*plugins.PluginAgent, llmClient, guidelines, cfg interface{}) *MCPInterface {
	var executor *plugins.PluginExecutor
	var promptLoader *prompts.Loader

	// Try to create prompt loader if config is available
	if cfg != nil {
		if config, ok := cfg.(*config.Config); ok && config.Agent.PromptsPath != "" {
			// Support comma-separated paths for multiple prompt locations
			paths := strings.Split(config.Agent.PromptsPath, ",")
			// Trim whitespace from each path
			for i, path := range paths {
				paths[i] = strings.TrimSpace(path)
			}

			// Also add agent-specific prompt directories
			if config.Agent.PluginsPath != "" {
				// Add custom agents prompts directory
				customPromptsPath := filepath.Join(config.Agent.PluginsPath, "custom", "prompts")
				paths = append(paths, customPromptsPath)

				// Add core agents prompts directory
				corePromptsPath := filepath.Join(config.Agent.PluginsPath, "core", "prompts")
				paths = append(paths, corePromptsPath)
			}

			loader, err := prompts.NewMultiPathLoader(paths)
			if err == nil {
				promptLoader = loader
			}
		}
	}

	if llmClient != nil {
		if llm, ok := llmClient.(*llm.Client); ok {
			executor = plugins.NewPluginExecutor(llm, ghClient, promptLoader)
		}
	}

	return &MCPInterface{
		githubClient:   ghClient,
		pluginAgents:   pluginAgents,
		pluginExecutor: executor,
		llmClient:      llmClient,
		guidelines:     guidelines,
		config:         cfg,
	}
}

// ExecuteAgent executes an agent by name with given parameters
func (m *MCPInterface) ExecuteAgent(ctx context.Context, agentName string, params map[string]interface{}) (interface{}, error) {
	// Find and execute plugin agent
	for _, pluginAgent := range m.pluginAgents {
		if pluginAgent.Name == agentName {
			if m.pluginExecutor != nil {
				return m.pluginExecutor.Execute(ctx, pluginAgent, params)
			}
			return nil, fmt.Errorf("plugin executor not available")
		}
	}

	return nil, fmt.Errorf("agent not found: %s", agentName)
}

// ExecuteWorkflow executes a workflow by name
// Note: Workflows are not yet supported in plugin-only mode
func (m *MCPInterface) ExecuteWorkflow(ctx context.Context, workflowName string, params map[string]interface{}) (interface{}, error) {
	return nil, fmt.Errorf("workflows not yet supported in plugin-only mode")
}

// GetAgentCapabilities returns the capabilities of a specific agent
func (m *MCPInterface) GetAgentCapabilities(agentName string) ([]string, error) {
	for _, pluginAgent := range m.pluginAgents {
		if pluginAgent.Name == agentName {
			return pluginAgent.Actions, nil
		}
	}
	return nil, fmt.Errorf("agent not found: %s", agentName)
}

// ListAgents returns all available agents
func (m *MCPInterface) ListAgents() []string {
	var agents []string
	for _, pluginAgent := range m.pluginAgents {
		agents = append(agents, pluginAgent.Name)
	}
	return agents
}

// ListWorkflows returns all available workflows
// Note: Workflows are not yet supported in plugin-only mode
func (m *MCPInterface) ListWorkflows() []string {
	return []string{}
}

// ToJSON converts the MCP interface state to JSON for external consumption
func (m *MCPInterface) ToJSON() ([]byte, error) {
	data := map[string]interface{}{
		"agents":    m.ListAgents(),
		"workflows": m.ListWorkflows(),
	}
	return json.MarshalIndent(data, "", "  ")
}
