package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"go-agent-cli/internal/config"
	"go-agent-cli/internal/generator"
)

var createCmd = &cobra.Command{
	Use:   "create [project-name]",
	Short: "Create a new Go agent project",
	Long: `Create a new Go agent project with the specified name and mode.

Modes:
  default  - Standard agent with skills system
  provider - Integration agent with external service adapters`,
	Args: cobra.ExactArgs(1),
	Run:  runCreate,
}

var (
	mode      string
	outputDir string
	module    string
)

func init() {
	createCmd.Flags().StringVarP(&mode, "mode", "m", "default", "Project mode (default|provider)")
	createCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "Output directory")
	createCmd.Flags().StringVar(&module, "module", "", "Go module name (default: project-name)")
}

func runCreate(cmd *cobra.Command, args []string) {
	projectName := args[0]
	
	// Validate mode
	if mode != "default" && mode != "provider" {
		fmt.Fprintf(os.Stderr, "Error: Invalid mode '%s'. Must be 'default' or 'provider'\n", mode)
		os.Exit(1)
	}

	// Set default module name if not provided
	if module == "" {
		module = projectName
	}

	// Create project config
	projectConfig := &config.ProjectConfig{
		Name:       projectName,
		ModuleName: module,
		Mode:       mode,
		OutputDir:  outputDir,
	}

	// Create template data
	templateData := &config.TemplateData{
		ProjectName: projectName,
		ModuleName:  module,
		Mode:        mode,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05"),
		AgentCard:   createDefaultAgentCard(projectName, mode),
	}

	// Create generator
	gen := generator.New()

	// Generate project
	projectPath := filepath.Join(outputDir, projectName)
	if err := gen.Generate(projectConfig, templateData, projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully created %s agent project: %s\n", mode, projectName)
	fmt.Printf("📁 Project location: %s\n", projectPath)
	fmt.Printf("🚀 Next steps:\n")
	fmt.Printf("   cd %s\n", projectName)
	fmt.Printf("   go mod tidy\n")
	fmt.Printf("   go run main.go\n")
}

func createDefaultAgentCard(projectName, mode string) config.AgentCard {
	card := config.AgentCard{
		Name:        projectName,
		Description: fmt.Sprintf("A %s mode Go agent", mode),
		URL:         "http://localhost:8080",
		Version:     "1.0.0",
		Capabilities: config.AgentCapabilities{
			SupportsStreaming: true,
			SupportedModes:    []string{"text", "json"},
		},
		DefaultInputModes:  []string{"text"},
		DefaultOutputModes: []string{"text"},
		Skills:             []config.AgentSkill{},
	}

	// Add mode-specific configuration
	if mode == "provider" {
		card.Provider = &config.AgentProvider{
			Name:        fmt.Sprintf("%s-provider", projectName),
			Description: "External service provider integration",
			URL:         "http://localhost:8080/provider",
		}
		
		card.Skills = append(card.Skills, config.AgentSkill{
			Name:        "external_service_call",
			Description: "Call external service through adapter",
			Parameters: map[string]interface{}{
				"endpoint": map[string]interface{}{
					"type":        "string",
					"description": "External service endpoint",
					"required":    true,
				},
			},
		})
	} else {
		card.Skills = append(card.Skills, config.AgentSkill{
			Name:        "example_skill",
			Description: "An example skill implementation",
			Parameters: map[string]interface{}{
				"input": map[string]interface{}{
					"type":        "string",
					"description": "Input text for processing",
					"required":    true,
				},
			},
		})
	}

	return card
}