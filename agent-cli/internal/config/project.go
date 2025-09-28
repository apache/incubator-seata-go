package config

type ProjectConfig struct {
	Name       string `json:"name"`
	ModuleName string `json:"moduleName"`
	Mode       string `json:"mode"` // "default" or "provider"
	OutputDir  string `json:"outputDir"`
}

type AgentCard struct {
	Name                              string                    `json:"name"`
	Description                       string                    `json:"description"`
	URL                               string                    `json:"url"`
	Provider                          *AgentProvider            `json:"provider,omitempty"`
	Version                           string                    `json:"version"`
	DocumentationURL                  *string                   `json:"documentationUrl,omitempty"`
	Capabilities                      AgentCapabilities         `json:"capabilities"`
	SecuritySchemes                   map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Security                          []map[string][]string     `json:"security,omitempty"`
	DefaultInputModes                 []string                  `json:"defaultInputModes"`
	DefaultOutputModes                []string                  `json:"defaultOutputModes"`
	Skills                            []AgentSkill              `json:"skills"`
	SupportsAuthenticatedExtendedCard *bool                     `json:"supportsAuthenticatedExtendedCard,omitempty"`
}

type AgentProvider struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type AgentCapabilities struct {
	SupportsStreaming bool     `json:"supportsStreaming"`
	SupportedModes    []string `json:"supportedModes"`
}

type SecurityScheme struct {
	Type         string            `json:"type"`
	Scheme       string            `json:"scheme,omitempty"`
	BearerFormat string            `json:"bearerFormat,omitempty"`
	Description  string            `json:"description,omitempty"`
	Flows        map[string]string `json:"flows,omitempty"`
}

type AgentSkill struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type TemplateData struct {
	ProjectName string
	ModuleName  string
	Mode        string
	Timestamp   string
	AgentCard   AgentCard
}
