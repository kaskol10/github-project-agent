package guidelines

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type Guidelines struct {
	RawContent    string
	FormatRules   FormatRules
	Instructions  string
	Examples      []Example
}

type FormatRules struct {
	RequiredSections     []string
	MinDescriptionLength int
	RequireLabels        bool
	LabelPrefix          string
	LabelRequirements    []LabelRequirement
}

type LabelRequirement struct {
	Type        string // "priority", "type", "team", etc.
	Required    bool
	AllowedValues []string // Optional: specific values allowed
}

type Example struct {
	Title       string
	Description string
	Good        string
	Bad         string
}

func LoadFromFile(filePath string) (*Guidelines, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read guidelines file: %w", err)
	}
	
	return Parse(string(content))
}

func LoadFromReader(reader io.Reader) (*Guidelines, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read guidelines: %w", err)
	}
	
	return Parse(string(content))
}

func Parse(content string) (*Guidelines, error) {
	g := &Guidelines{
		RawContent: content,
		FormatRules: FormatRules{
			RequiredSections:     []string{},
			MinDescriptionLength: 50,
			RequireLabels:        false,
			LabelPrefix:          "priority:",
			LabelRequirements:    []LabelRequirement{},
		},
		Examples: []Example{},
	}
	
	// Extract format rules
	g.extractFormatRules(content)
	
	// Extract instructions
	g.extractInstructions(content)
	
	// Extract examples
	g.extractExamples(content)
	
	return g, nil
}

func (g *Guidelines) extractFormatRules(content string) {
	// Look for "## Format Rules" or "## Format Requirements" section
	formatSection := extractSection(content, "Format Rules", "Format Requirements", "Format")
	if formatSection == "" {
		return
	}
	
	// Extract required sections
	requiredSections := extractListItems(formatSection, "Required Sections", "Required sections", "Sections")
	if len(requiredSections) > 0 {
		g.FormatRules.RequiredSections = requiredSections
	}
	
	// Extract minimum description length
	minLength := extractIntValue(formatSection, "Minimum.*length", "Min.*length", "Description.*length")
	if minLength > 0 {
		g.FormatRules.MinDescriptionLength = minLength
	}
	
	// Extract label requirements
	if strings.Contains(strings.ToLower(formatSection), "label") {
		g.FormatRules.RequireLabels = true
		
		// Extract label prefix
		prefix := extractStringValue(formatSection, "label.*prefix", "prefix.*label")
		if prefix != "" {
			g.FormatRules.LabelPrefix = prefix
		}
		
		// Extract label requirements
		labelReqs := extractLabelRequirements(formatSection)
		if len(labelReqs) > 0 {
			g.FormatRules.LabelRequirements = labelReqs
		}
	}
}

func (g *Guidelines) extractInstructions(content string) {
	// Extract main instructions (everything that's not format rules or examples)
	sections := []string{
		extractSection(content, "Instructions", "Guidelines", "Guidelines and Rules"),
		extractSection(content, "General", "Overview"),
	}
	
	var instructions []string
	for _, section := range sections {
		if section != "" {
			instructions = append(instructions, section)
		}
	}
	
	g.Instructions = strings.Join(instructions, "\n\n")
}

func (g *Guidelines) extractExamples(content string) {
	// Look for examples section
	examplesSection := extractSection(content, "Examples", "Example Tasks", "Good Examples")
	if examplesSection == "" {
		return
	}
	
	// Simple extraction: look for code blocks or quoted sections
	// Use [\s\S] to match any character including newlines (Go regexp doesn't support (?s))
	codeBlockPattern := regexp.MustCompile("```[\\w]*\\n([\\s\\S]*?)```")
	codeBlocks := codeBlockPattern.FindAllStringSubmatch(examplesSection, -1)
	
	for i, block := range codeBlocks {
		if i < len(codeBlocks)-1 {
			ex := Example{
				Description: fmt.Sprintf("Example %d", i+1),
				Good:        block[1],
			}
			g.Examples = append(g.Examples, ex)
		}
	}
}

// Helper functions

func extractSection(content string, titles ...string) string {
	lines := strings.Split(content, "\n")
	
	for _, title := range titles {
		// Find the section header (case-insensitive)
		headerPattern := regexp.MustCompile(fmt.Sprintf(`(?i)^##+\s*%s\s*$`, regexp.QuoteMeta(title)))
		
		var startIdx = -1
		for i, line := range lines {
			if headerPattern.MatchString(line) {
				startIdx = i + 1
				break
			}
		}
		
		if startIdx == -1 {
			continue
		}
		
		// Find the end of the section (next ## header or end of content)
		var endIdx = len(lines)
		for i := startIdx; i < len(lines); i++ {
			if strings.HasPrefix(strings.TrimSpace(lines[i]), "##") {
				endIdx = i
				break
			}
		}
		
		// Extract the section content
		if startIdx < endIdx {
			sectionLines := lines[startIdx:endIdx]
			return strings.TrimSpace(strings.Join(sectionLines, "\n"))
		}
	}
	return ""
}

func extractListItems(section string, keywords ...string) []string {
	var items []string
	
	for _, keyword := range keywords {
		pattern := regexp.MustCompile(fmt.Sprintf(`(?i)%s[:\s]*\n((?:[-*]\s+.*\n?)+)`, regexp.QuoteMeta(keyword)))
		matches := pattern.FindStringSubmatch(section)
		if len(matches) > 1 {
			lines := strings.Split(matches[1], "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				// Remove list markers
				line = regexp.MustCompile(`^[-*]\s+`).ReplaceAllString(line, "")
				if line != "" {
					items = append(items, line)
				}
			}
			break
		}
	}
	
	return items
}

func extractIntValue(section string, patterns ...string) int {
	for _, pattern := range patterns {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)%s[:\s]*(\d+)`, pattern))
		matches := re.FindStringSubmatch(section)
		if len(matches) > 1 {
			var value int
			if _, err := fmt.Sscanf(matches[1], "%d", &value); err == nil {
				return value
			}
		}
	}
	return 0
}

func extractStringValue(section string, patterns ...string) string {
	for _, pattern := range patterns {
		re := regexp.MustCompile(fmt.Sprintf(`(?i)%s[:\s]*["']?([^"'\n]+)["']?`, pattern))
		matches := re.FindStringSubmatch(section)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}

func extractLabelRequirements(section string) []LabelRequirement {
	var reqs []LabelRequirement
	
	// Look for label requirements in various formats
	labelPattern := regexp.MustCompile(`(?i)(?:label|tag)[:\s]+(priority|type|team|status)[:\s]+(required|optional)?[:\s]*(.*)`)
	matches := labelPattern.FindAllStringSubmatch(section, -1)
	
	for _, match := range matches {
		if len(match) >= 2 {
			req := LabelRequirement{
				Type:     strings.ToLower(match[1]),
				Required: strings.Contains(strings.ToLower(match[2]), "required"),
			}
			
			if len(match) > 3 && match[3] != "" {
				// Extract allowed values
				values := strings.Split(match[3], ",")
				for _, v := range values {
					v = strings.TrimSpace(v)
					if v != "" {
						req.AllowedValues = append(req.AllowedValues, v)
					}
				}
			}
			
			reqs = append(reqs, req)
		}
	}
	
	return reqs
}

