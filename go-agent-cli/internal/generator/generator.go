package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"go-agent-cli/internal/config"
)

type Generator struct {
	templates map[string]*template.Template
}

func New() *Generator {
	return &Generator{
		templates: make(map[string]*template.Template),
	}
}

func (g *Generator) Generate(projectConfig *config.ProjectConfig, templateData *config.TemplateData, outputPath string) error {
	// Create output directory
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate project structure based on mode
	switch projectConfig.Mode {
	case "default":
		return g.generateDefaultProject(templateData, outputPath)
	case "provider":
		return g.generateProviderProject(templateData, outputPath)
	default:
		return fmt.Errorf("unsupported mode: %s", projectConfig.Mode)
	}
}

func (g *Generator) generateDefaultProject(data *config.TemplateData, outputPath string) error {
	// Create directory structure
	dirs := []string{
		"config",
		"internal/agent",
		"internal/agent/skills",
		"internal/agent/handlers",
		"internal/config",
		"internal/utils",
		"scripts",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(outputPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		"main.go":                          defaultMainTemplate,
		"go.mod":                           goModTemplate,
		"config/config.yaml":               defaultConfigTemplate,
		"internal/agent/agent.go":          defaultAgentTemplate,
		"internal/agent/skills/example.go": defaultSkillTemplate,
		"internal/agent/handlers/rpc.go":   defaultRPCTemplate,
		"internal/config/config.go":        configLoaderTemplate,
		"internal/utils/logger.go":         loggerTemplate,
		"scripts/build.sh":                 buildScriptTemplate,
		"scripts/run.sh":                   runScriptTemplate,
		"Dockerfile":                       dockerfileTemplate,
		"docker-compose.yml":               dockerComposeTemplate,
		"README.md":                        defaultReadmeTemplate,
	}

	for filePath, tmplContent := range files {
		if err := g.generateFile(filepath.Join(outputPath, filePath), tmplContent, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filePath, err)
		}
	}

	// Generate agent_card.json
	return g.generateAgentCard(filepath.Join(outputPath, "config/agent_card.json"), &data.AgentCard)
}

func (g *Generator) generateProviderProject(data *config.TemplateData, outputPath string) error {
	// Create directory structure
	dirs := []string{
		"config",
		"internal/agent",
		"internal/agent/provider",
		"internal/skills",
		"internal/handlers",
		"examples",
		"docs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(outputPath, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Generate files
	files := map[string]string{
		"main.go":                              providerMainTemplate,
		"go.mod":                               goModTemplate,
		"config/config.yaml":                   providerConfigTemplate,
		"internal/agent/agent.go":              providerAgentTemplate,
		"internal/agent/provider/adapter.go":   providerAdapterTemplate,
		"internal/agent/provider/client.go":    providerClientTemplate,
		"internal/agent/provider/mapper.go":    providerMapperTemplate,
		"internal/skills/provider_skills.go":   providerSkillsTemplate,
		"internal/handlers/rpc.go":             providerRPCTemplate,
		"examples/external_service.go":         externalServiceTemplate,
		"docs/integration.md":                  integrationDocTemplate,
	}

	for filePath, tmplContent := range files {
		if err := g.generateFile(filepath.Join(outputPath, filePath), tmplContent, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filePath, err)
		}
	}

	// Generate agent_card.json
	return g.generateAgentCard(filepath.Join(outputPath, "config/agent_card.json"), &data.AgentCard)
}

func (g *Generator) generateFile(filePath, tmplContent string, data *config.TemplateData) error {
	// Create directory if not exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Parse template
	tmpl, err := template.New(filepath.Base(filePath)).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Execute template
	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

func (g *Generator) generateAgentCard(filePath string, agentCard *config.AgentCard) error {
	// Create directory if not exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create agent card file: %w", err)
	}
	defer file.Close()

	// Marshal to JSON with indentation
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(agentCard); err != nil {
		return fmt.Errorf("failed to encode agent card: %w", err)
	}

	return nil
}