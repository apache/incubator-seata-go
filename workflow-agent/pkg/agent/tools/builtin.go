/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tools

import (
	"context"
	"fmt"
	"seata-go-ai-workflow-agent/pkg/agent/types"
	"strings"
	"time"
)

// RegisterBuiltinTools registers all built-in tools to the registry
func RegisterBuiltinTools(registry types.ToolRegistry) error {
	tools := []*types.Tool{
		NewEchoTool(),
		NewCalculatorTool(),
		NewDateTimeTool(),
		NewStringTool(),
		NewWaitTool(),
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool '%s': %w", tool.Name, err)
		}
	}

	return nil
}

// NewEchoTool creates a simple echo tool for testing
func NewEchoTool() *types.Tool {
	return &types.Tool{
		Name:        "echo",
		Description: "Echoes back the input message",
		Parameters: map[string]types.Parameter{
			"message": {
				Type:        "string",
				Description: "Message to echo back",
				Required:    true,
			},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			message, ok := input["message"].(string)
			if !ok {
				return nil, fmt.Errorf("message parameter must be a string")
			}
			return fmt.Sprintf("Echo: %s", message), nil
		},
	}
}

// NewCalculatorTool creates a basic calculator tool
func NewCalculatorTool() *types.Tool {
	return &types.Tool{
		Name:        "calculator",
		Description: "Performs basic arithmetic operations",
		Parameters: map[string]types.Parameter{
			"operation": {
				Type:        "string",
				Description: "Operation to perform",
				Required:    true,
				Enum:        []string{"add", "subtract", "multiply", "divide"},
			},
			"a": {
				Type:        "number",
				Description: "First number",
				Required:    true,
			},
			"b": {
				Type:        "number",
				Description: "Second number",
				Required:    true,
			},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			operation, ok := input["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter must be a string")
			}

			a, err := getNumber(input["a"])
			if err != nil {
				return nil, fmt.Errorf("parameter 'a' must be a number: %w", err)
			}

			b, err := getNumber(input["b"])
			if err != nil {
				return nil, fmt.Errorf("parameter 'b' must be a number: %w", err)
			}

			var result float64
			switch operation {
			case "add":
				result = a + b
			case "subtract":
				result = a - b
			case "multiply":
				result = a * b
			case "divide":
				if b == 0 {
					return nil, fmt.Errorf("division by zero")
				}
				result = a / b
			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}

			return map[string]interface{}{
				"result":    result,
				"operation": operation,
				"a":         a,
				"b":         b,
			}, nil
		},
	}
}

// NewDateTimeTool creates a date/time tool
func NewDateTimeTool() *types.Tool {
	return &types.Tool{
		Name:        "datetime",
		Description: "Gets current date and time information",
		Parameters: map[string]types.Parameter{
			"format": {
				Type:        "string",
				Description: "Time format (iso, unix, custom)",
				Required:    false,
				Default:     "iso",
				Enum:        []string{"iso", "unix", "rfc3339", "custom"},
			},
			"timezone": {
				Type:        "string",
				Description: "Timezone (UTC, Local, or IANA timezone)",
				Required:    false,
				Default:     "UTC",
			},
			"custom_format": {
				Type:        "string",
				Description: "Custom time format (Go time format)",
				Required:    false,
			},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			format := "iso"
			if f, ok := input["format"].(string); ok {
				format = f
			}

			timezone := "UTC"
			if tz, ok := input["timezone"].(string); ok {
				timezone = tz
			}

			now := time.Now()

			// Handle timezone
			var loc *time.Location
			var err error
			switch timezone {
			case "UTC":
				loc = time.UTC
			case "Local":
				loc = time.Local
			default:
				loc, err = time.LoadLocation(timezone)
				if err != nil {
					return nil, fmt.Errorf("invalid timezone: %s", timezone)
				}
			}

			now = now.In(loc)

			var timeStr string
			switch format {
			case "iso":
				timeStr = now.Format(time.RFC3339)
			case "unix":
				timeStr = fmt.Sprintf("%d", now.Unix())
			case "rfc3339":
				timeStr = now.Format(time.RFC3339)
			case "custom":
				customFormat, ok := input["custom_format"].(string)
				if !ok {
					return nil, fmt.Errorf("custom_format required when format is 'custom'")
				}
				timeStr = now.Format(customFormat)
			default:
				return nil, fmt.Errorf("unsupported format: %s", format)
			}

			return map[string]interface{}{
				"timestamp": timeStr,
				"timezone":  timezone,
				"format":    format,
				"unix":      now.Unix(),
				"year":      now.Year(),
				"month":     int(now.Month()),
				"day":       now.Day(),
				"hour":      now.Hour(),
				"minute":    now.Minute(),
				"second":    now.Second(),
				"weekday":   now.Weekday().String(),
			}, nil
		},
	}
}

// NewStringTool creates a string manipulation tool
func NewStringTool() *types.Tool {
	return &types.Tool{
		Name:        "string",
		Description: "Performs string operations",
		Parameters: map[string]types.Parameter{
			"operation": {
				Type:        "string",
				Description: "String operation to perform",
				Required:    true,
				Enum:        []string{"upper", "lower", "length", "contains", "split", "join", "trim"},
			},
			"text": {
				Type:        "string",
				Description: "Input text",
				Required:    true,
			},
			"separator": {
				Type:        "string",
				Description: "Separator for split/join operations",
				Required:    false,
				Default:     ",",
			},
			"substring": {
				Type:        "string",
				Description: "Substring for contains operation",
				Required:    false,
			},
			"array": {
				Type:        "array",
				Description: "Array of strings for join operation",
				Required:    false,
			},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			operation, ok := input["operation"].(string)
			if !ok {
				return nil, fmt.Errorf("operation parameter must be a string")
			}

			text, ok := input["text"].(string)
			if !ok && operation != "join" {
				return nil, fmt.Errorf("text parameter must be a string")
			}

			separator := ","
			if s, ok := input["separator"].(string); ok {
				separator = s
			}

			switch operation {
			case "upper":
				return strings.ToUpper(text), nil

			case "lower":
				return strings.ToLower(text), nil

			case "length":
				return len(text), nil

			case "contains":
				substring, ok := input["substring"].(string)
				if !ok {
					return nil, fmt.Errorf("substring parameter required for contains operation")
				}
				return strings.Contains(text, substring), nil

			case "split":
				return strings.Split(text, separator), nil

			case "join":
				arrayInterface, ok := input["array"]
				if !ok {
					return nil, fmt.Errorf("array parameter required for join operation")
				}

				array, ok := arrayInterface.([]interface{})
				if !ok {
					return nil, fmt.Errorf("array parameter must be an array")
				}

				strArray := make([]string, len(array))
				for i, item := range array {
					str, ok := item.(string)
					if !ok {
						return nil, fmt.Errorf("array item %d must be a string", i)
					}
					strArray[i] = str
				}

				return strings.Join(strArray, separator), nil

			case "trim":
				return strings.TrimSpace(text), nil

			default:
				return nil, fmt.Errorf("unsupported operation: %s", operation)
			}
		},
	}
}

// NewWaitTool creates a wait/delay tool
func NewWaitTool() *types.Tool {
	return &types.Tool{
		Name:        "wait",
		Description: "Waits for a specified amount of time",
		Parameters: map[string]types.Parameter{
			"duration": {
				Type:        "number",
				Description: "Duration to wait in seconds",
				Required:    true,
			},
			"message": {
				Type:        "string",
				Description: "Optional message to return after waiting",
				Required:    false,
				Default:     "Wait completed",
			},
		},
		Handler: func(ctx context.Context, input map[string]interface{}) (interface{}, error) {
			duration, err := getNumber(input["duration"])
			if err != nil {
				return nil, fmt.Errorf("duration parameter must be a number: %w", err)
			}

			if duration < 0 {
				return nil, fmt.Errorf("duration must be positive")
			}

			if duration > 30 {
				return nil, fmt.Errorf("maximum wait duration is 30 seconds")
			}

			message := "Wait completed"
			if m, ok := input["message"].(string); ok {
				message = m
			}

			waitDuration := time.Duration(duration * float64(time.Second))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitDuration):
				return map[string]interface{}{
					"message":  message,
					"duration": duration,
					"waited":   waitDuration.String(),
				}, nil
			}
		},
	}
}

// getNumber converts interface{} to float64
func getNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("value is not a number")
	}
}
