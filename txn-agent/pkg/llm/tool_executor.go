package llm

import (
	"context"
	"fmt"
	"time"

	"seata-go-ai-txn-agent/pkg/utils"
)

// AdvancedToolExecutor provides enhanced tool execution with error handling, logging, and metrics
type AdvancedToolExecutor struct {
	executor ToolExecutor
	logger   *utils.Logger
	timeout  time.Duration
	retries  int
}

// NewAdvancedToolExecutor creates a new advanced tool executor wrapper
func NewAdvancedToolExecutor(executor ToolExecutor, timeout time.Duration, retries int) *AdvancedToolExecutor {
	logger := utils.GetLogger("llm.advanced_tool_executor")

	return &AdvancedToolExecutor{
		executor: executor,
		logger:   logger,
		timeout:  timeout,
		retries:  retries,
	}
}

// Execute runs the tool with enhanced error handling and retries
func (a *AdvancedToolExecutor) Execute(ctx context.Context, toolCall ToolCall) (string, error) {
	// Create timeout context
	if a.timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, a.timeout)
		defer cancel()
		ctx = timeoutCtx
	}

	startTime := time.Now()
	a.logger.WithFields(map[string]interface{}{
		"tool_name": toolCall.Function.Name,
		"tool_id":   toolCall.ID,
		"arguments": toolCall.Function.Arguments,
	}).Info("Starting tool execution")

	var lastErr error

	// Retry logic
	for attempt := 0; attempt <= a.retries; attempt++ {
		if attempt > 0 {
			a.logger.WithFields(map[string]interface{}{
				"tool_name": toolCall.Function.Name,
				"attempt":   attempt,
				"error":     lastErr.Error(),
			}).Warn("Retrying tool execution")

			// Add exponential backoff
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		result, err := a.executor.Execute(ctx, toolCall)

		duration := time.Since(startTime)

		if err == nil {
			a.logger.WithFields(map[string]interface{}{
				"tool_name": toolCall.Function.Name,
				"tool_id":   toolCall.ID,
				"duration":  duration,
				"attempt":   attempt + 1,
			}).Info("Tool execution completed successfully")

			return result, nil
		}

		lastErr = err

		// Don't retry certain types of errors
		if !isRetryableToolError(err) {
			break
		}
	}

	duration := time.Since(startTime)
	a.logger.WithFields(map[string]interface{}{
		"tool_name": toolCall.Function.Name,
		"tool_id":   toolCall.ID,
		"duration":  duration,
		"attempts":  a.retries + 1,
		"error":     lastErr.Error(),
	}).Error("Tool execution failed after all retries")

	return "", fmt.Errorf("tool execution failed after %d attempts: %w", a.retries+1, lastErr)
}

// isRetryableToolError determines if a tool execution error should be retried
func isRetryableToolError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	// Don't retry validation errors or argument parsing errors
	if contains(errMsg, "failed to parse") ||
		contains(errMsg, "invalid argument") ||
		contains(errMsg, "required field") ||
		contains(errMsg, "division by zero") {
		return false
	}

	// Retry network errors, timeouts, temporary failures
	return contains(errMsg, "timeout") ||
		contains(errMsg, "connection") ||
		contains(errMsg, "temporary") ||
		contains(errMsg, "unavailable")
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(substr) == 0 ||
			(len(s) > 0 && s[0:len(substr)] == substr) ||
			(len(s) > len(substr) && contains(s[1:], substr)))
}

// ToolExecutionResult represents the result of tool execution with metadata
type ToolExecutionResult struct {
	ToolCall  ToolCall      `json:"tool_call"`
	Result    string        `json:"result"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Attempts  int           `json:"attempts"`
	Timestamp time.Time     `json:"timestamp"`
	Success   bool          `json:"success"`
}

// ParallelToolExecutor executes multiple tools in parallel
type ParallelToolExecutor struct {
	registry ToolRegistry
	logger   *utils.Logger
	timeout  time.Duration
	retries  int
}

// NewParallelToolExecutor creates a new parallel tool executor
func NewParallelToolExecutor(registry ToolRegistry, timeout time.Duration, retries int) *ParallelToolExecutor {
	logger := utils.GetLogger("llm.parallel_tool_executor")

	return &ParallelToolExecutor{
		registry: registry,
		logger:   logger,
		timeout:  timeout,
		retries:  retries,
	}
}

// ExecuteParallel executes multiple tool calls in parallel
func (p *ParallelToolExecutor) ExecuteParallel(ctx context.Context, toolCalls []ToolCall) []ToolExecutionResult {
	results := make([]ToolExecutionResult, len(toolCalls))
	resultChan := make(chan struct {
		index  int
		result ToolExecutionResult
	}, len(toolCalls))

	// Start goroutines for each tool call
	for i, toolCall := range toolCalls {
		go func(index int, tc ToolCall) {
			startTime := time.Now()

			executor, exists := p.registry.GetExecutor(tc.Function.Name)
			if !exists {
				resultChan <- struct {
					index  int
					result ToolExecutionResult
				}{
					index: index,
					result: ToolExecutionResult{
						ToolCall:  tc,
						Error:     fmt.Sprintf("Tool %s not found", tc.Function.Name),
						Duration:  time.Since(startTime),
						Attempts:  0,
						Timestamp: time.Now(),
						Success:   false,
					},
				}
				return
			}

			// Wrap with advanced executor
			advancedExecutor := NewAdvancedToolExecutor(executor, p.timeout, p.retries)

			result, err := advancedExecutor.Execute(ctx, tc)

			execResult := ToolExecutionResult{
				ToolCall:  tc,
				Duration:  time.Since(startTime),
				Timestamp: time.Now(),
				Attempts:  p.retries + 1,
			}

			if err != nil {
				execResult.Error = err.Error()
				execResult.Success = false
			} else {
				execResult.Result = result
				execResult.Success = true
			}

			resultChan <- struct {
				index  int
				result ToolExecutionResult
			}{
				index:  index,
				result: execResult,
			}
		}(i, toolCall)
	}

	// Collect results
	for i := 0; i < len(toolCalls); i++ {
		select {
		case result := <-resultChan:
			results[result.index] = result.result
		case <-ctx.Done():
			// Handle timeout for remaining tools
			for j := i; j < len(toolCalls); j++ {
				if results[j].Timestamp.IsZero() {
					results[j] = ToolExecutionResult{
						ToolCall:  toolCalls[j],
						Error:     "Context cancelled or timeout",
						Duration:  0,
						Attempts:  0,
						Timestamp: time.Now(),
						Success:   false,
					}
				}
			}
			break
		}
	}

	return results
}

// ToolCallBatch represents a batch of tool calls to be executed
type ToolCallBatch struct {
	ID        string     `json:"id"`
	ToolCalls []ToolCall `json:"tool_calls"`
	CreatedAt time.Time  `json:"created_at"`
}

// BatchToolExecutor manages batched tool execution
type BatchToolExecutor struct {
	parallelExecutor *ParallelToolExecutor
	logger           *utils.Logger
	maxBatchSize     int
}

// NewBatchToolExecutor creates a new batch tool executor
func NewBatchToolExecutor(registry ToolRegistry, timeout time.Duration, retries int, maxBatchSize int) *BatchToolExecutor {
	return &BatchToolExecutor{
		parallelExecutor: NewParallelToolExecutor(registry, timeout, retries),
		logger:           utils.GetLogger("llm.bash_tool"),
		maxBatchSize:     maxBatchSize,
	}
}

// ExecuteBatch executes a batch of tool calls
func (b *BatchToolExecutor) ExecuteBatch(ctx context.Context, batch ToolCallBatch) []ToolExecutionResult {
	b.logger.WithFields(map[string]interface{}{
		"batch_id":   batch.ID,
		"tool_count": len(batch.ToolCalls),
		"created_at": batch.CreatedAt,
	}).Info("Starting batch tool execution")

	// Split into smaller batches if needed
	if len(batch.ToolCalls) > b.maxBatchSize {
		var allResults []ToolExecutionResult

		for i := 0; i < len(batch.ToolCalls); i += b.maxBatchSize {
			end := i + b.maxBatchSize
			if end > len(batch.ToolCalls) {
				end = len(batch.ToolCalls)
			}

			subBatch := batch.ToolCalls[i:end]
			results := b.parallelExecutor.ExecuteParallel(ctx, subBatch)
			allResults = append(allResults, results...)
		}

		return allResults
	}

	return b.parallelExecutor.ExecuteParallel(ctx, batch.ToolCalls)
}
