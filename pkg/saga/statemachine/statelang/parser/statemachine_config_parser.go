package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
)

type ConfigParser interface {
	Parse(configContent []byte) (*StateMachineObject, error)
}

type StateMachineConfigParser struct{}

func NewStateMachineConfigParser() *StateMachineConfigParser {
	return &StateMachineConfigParser{}
}

func (p *StateMachineConfigParser) checkConfigFile(configFilePath string) error {
	if _, err := os.Stat(configFilePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("config file %s does not exist: %w", configFilePath, err)
		}
		return fmt.Errorf("failed to access config file %s: %w", configFilePath, err)
	}
	return nil
}

func (p *StateMachineConfigParser) readFile(configFilePath string) ([]byte, error) {
	file, _ := os.Open(configFilePath)
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	var buf bytes.Buffer
	_, err := io.Copy(&buf, file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFilePath, err)
	}

	return buf.Bytes(), nil
}

func (p *StateMachineConfigParser) getParser(configFilePath string) (ConfigParser, error) {
	fileExt := filepath.Ext(configFilePath)
	switch fileExt {
	case ".json":
		return NewJSONConfigParser(), nil
	case ".yaml", ".yml":
		return NewYAMLConfigParser(), nil
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", fileExt)
	}
}

func (p *StateMachineConfigParser) Parse(configFilePath string) (*StateMachineObject, error) {
	if err := p.checkConfigFile(configFilePath); err != nil {
		return nil, err
	}

	configContent, err := p.readFile(configFilePath)
	if err != nil {
		return nil, err
	}

	parser, err := p.getParser(configFilePath)
	if err != nil {
		return nil, err
	}

	return parser.Parse(configContent)
}

type JSONConfigParser struct{}

func NewJSONConfigParser() *JSONConfigParser {
	return &JSONConfigParser{}
}

func (p *JSONConfigParser) Parse(configContent []byte) (*StateMachineObject, error) {
	if configContent == nil || len(configContent) == 0 {
		return nil, fmt.Errorf("empty JSON config content")
	}

	var stateMachineObject StateMachineObject
	if err := json.Unmarshal(configContent, &stateMachineObject); err != nil {
		return nil, fmt.Errorf("failed to parse JSON config content: %w", err)
	}

	return &stateMachineObject, nil
}

type YAMLConfigParser struct{}

func NewYAMLConfigParser() *YAMLConfigParser {
	return &YAMLConfigParser{}
}

func (p *YAMLConfigParser) Parse(configContent []byte) (*StateMachineObject, error) {
	if configContent == nil || len(configContent) == 0 {
		return nil, fmt.Errorf("empty YAML config content")
	}

	var stateMachineObject StateMachineObject
	if err := yaml.Unmarshal(configContent, &stateMachineObject); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config content: %w", err)
	}

	return &stateMachineObject, nil
}

type StateMachineObject struct {
	Name                        string                 `json:"Name" yaml:"Name"`
	Comment                     string                 `json:"Comment" yaml:"Comment"`
	Version                     string                 `json:"Version" yaml:"Version"`
	StartState                  string                 `json:"StartState" yaml:"StartState"`
	RecoverStrategy             string                 `json:"RecoverStrategy" yaml:"RecoverStrategy"`
	Persist                     bool                   `json:"IsPersist" yaml:"IsPersist"`
	RetryPersistModeUpdate      bool                   `json:"IsRetryPersistModeUpdate" yaml:"IsRetryPersistModeUpdate"`
	CompensatePersistModeUpdate bool                   `json:"IsCompensatePersistModeUpdate" yaml:"IsCompensatePersistModeUpdate"`
	Type                        string                 `json:"Type" yaml:"Type"`
	States                      map[string]interface{} `json:"States" yaml:"States"`
}
