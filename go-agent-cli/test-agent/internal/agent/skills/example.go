package skills

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type Skill interface {
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	Name() string
	Description() string
}

type ExampleSkill struct {
	logger *zap.Logger
}

func NewExampleSkill(logger *zap.Logger) *ExampleSkill {
	return &ExampleSkill{logger: logger}
}

func (s *ExampleSkill) Name() string {
	return "example"
}

func (s *ExampleSkill) Description() string {
	return "An example skill that processes input text"
}

func (s *ExampleSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	input, ok := params["input"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'input' parameter")
	}

	s.logger.Info("Executing example skill", zap.String("input", input))

	result := fmt.Sprintf("Processed: %s (length: %d)", input, len(input))

	return map[string]interface{}{
		"result":   result,
		"original": input,
		"length":   len(input),
	}, nil
}
