package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"seata-go-ai-txn-agent/pkg/llm"
	"seata-go-ai-txn-agent/pkg/utils"
)

// ReactFlowPosition represents position coordinates in React Flow
type ReactFlowPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ReactFlowNodeData represents the data object for React Flow nodes
type ReactFlowNodeData struct {
	Label string `json:"label"`
}

// ReactFlowNode represents a node in React Flow graph
type ReactFlowNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Position ReactFlowPosition      `json:"position"`
	Data     ReactFlowNodeData      `json:"data"`
	Style    map[string]interface{} `json:"style,omitempty"`
}

// ReactFlowEdge represents an edge in React Flow graph
type ReactFlowEdge struct {
	ID     string                 `json:"id"`
	Source string                 `json:"source"`
	Target string                 `json:"target"`
	Type   string                 `json:"type,omitempty"`
	Label  string                 `json:"label,omitempty"`
	Style  map[string]interface{} `json:"style,omitempty"`
}

// ReactFlowGraph represents the complete graph structure for React Flow
type ReactFlowGraph struct {
	Nodes []ReactFlowNode `json:"nodes"`
	Edges []ReactFlowEdge `json:"edges"`
}

// SeataJSON represents the Seata Saga workflow JSON
type SeataJSON struct {
	Name                          string                 `json:"Name"`
	Comment                       string                 `json:"Comment"`
	StartState                    string                 `json:"StartState"`
	Version                       string                 `json:"Version"`
	States                        map[string]interface{} `json:"States"`
	IsRetryPersistModeUpdate      bool                   `json:"IsRetryPersistModeUpdate,omitempty"`
	IsCompensatePersistModeUpdate bool                   `json:"IsCompensatePersistModeUpdate,omitempty"`
}

// AgentResponse represents the structured response from the agent
type AgentResponse struct {
	Text      string         `json:"text"`
	Graph     ReactFlowGraph `json:"graph"`
	SeataJSON *SeataJSON     `json:"seata_json"`
	Phase     int            `json:"phase"`
}

// ConversationMessage represents a message in the conversation history
type ConversationMessage struct {
	Role      llm.MessageRole `json:"role"`
	Content   string          `json:"content"`
	Timestamp time.Time       `json:"timestamp"`
}

// SagaWorkflowAgent represents the main agent for Seata Saga workflow generation
type SagaWorkflowAgent struct {
	client       llm.LLMClient
	systemPrompt string
	messages     []ConversationMessage
	logger       *utils.Logger
	mu           sync.RWMutex
}

// AgentConfig represents configuration for the agent
type AgentConfig struct {
	BaseURL    string
	APIKey     string
	Model      string
	Timeout    int
	MaxRetries int
}

// NewSagaWorkflowAgent creates a new Seata Saga workflow agent
func NewSagaWorkflowAgent(config AgentConfig) (*SagaWorkflowAgent, error) {
	logger := utils.GetLogger("saga-agent")

	llmConfig := llm.NewConfigBuilder().
		WithBaseURL(config.BaseURL).
		WithAPIKey(config.APIKey).
		WithModel(config.Model).
		WithTimeout(config.Timeout).
		WithMaxRetry(config.MaxRetries).
		Build()

	client, err := llm.NewClient(llm.ClientTypeOpenAI, llmConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	agent := &SagaWorkflowAgent{
		client:   client,
		messages: make([]ConversationMessage, 0),
		logger:   logger,
	}

	if err := agent.loadSystemPrompt(); err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	logger.Info("Saga workflow agent initialized successfully")
	return agent, nil
}

// loadSystemPrompt loads the system prompt from multiple possible locations
func (a *SagaWorkflowAgent) loadSystemPrompt() error {
	paths := []string{
		"./prompt/system_prompt.md",
		"../prompt/system_prompt.md",
		"../../prompt/system_prompt.md",
		"../../../prompt/system_prompt.md",
		"/home/finntew/GolandProjects/workflow-agent/prompt/system_prompt.md",
	}

	var wg sync.WaitGroup
	var once sync.Once
	var success bool
	done := make(chan struct{})

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			file, err := os.Open(p)
			if err != nil {
				return
			}
			defer file.Close()

			content, err := io.ReadAll(file)
			if err != nil {
				return
			}

			once.Do(func() {
				a.systemPrompt = string(content)
				success = true
				close(done)
			})
		}(path)
	}

	go func() {
		wg.Wait()
		if !success {
			close(done)
		}
	}()

	<-done

	if !success {
		return fmt.Errorf("failed to load system prompt from any location")
	}

	a.logger.Debug("System prompt loaded successfully")
	return nil
}

// SendMessage processes a user message and returns the agent's structured response
func (a *SagaWorkflowAgent) SendMessage(ctx context.Context, userMessage string) (*AgentResponse, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message to conversation history
	a.messages = append(a.messages, ConversationMessage{
		Role:      llm.RoleUser,
		Content:   userMessage,
		Timestamp: time.Now(),
	})

	// Build messages for LLM
	llmMessages := a.buildLLMMessages()

	// Prepare completion request
	req := &llm.CompletionRequest{
		Model:        a.client.GetConfig().Model,
		Messages:     llmMessages,
		SystemPrompt: a.systemPrompt,
		Temperature:  0.7,
		MaxTokens:    8192,
	}

	// Get structured response from LLM
	var response AgentResponse
	err := a.client.CompleteStructured(ctx, req, &response)
	if err != nil {
		a.logger.WithError(err).Error("Failed to get structured response from LLM")
		return nil, fmt.Errorf("failed to get response from LLM: %w", err)
	}

	data, _ := json.Marshal(response)

	// Add assistant response to conversation history
	a.messages = append(a.messages, ConversationMessage{
		Role:      llm.RoleAssistant,
		Content:   string(data),
		Timestamp: time.Now(),
	})

	a.logger.WithField("phase", response.Phase).Info("Generated response")
	return &response, nil
}

// buildLLMMessages converts conversation history to LLM messages
func (a *SagaWorkflowAgent) buildLLMMessages() []llm.Message {
	messages := make([]llm.Message, len(a.messages))
	for i, msg := range a.messages {
		messages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return messages
}

// GetConversationHistory returns the current conversation history
func (a *SagaWorkflowAgent) GetConversationHistory() []ConversationMessage {
	a.mu.RLock()
	defer a.mu.RUnlock()

	history := make([]ConversationMessage, len(a.messages))
	copy(history, a.messages)
	return history
}

// ClearConversation clears the conversation history
func (a *SagaWorkflowAgent) ClearConversation() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.messages = make([]ConversationMessage, 0)
	a.logger.Info("Conversation history cleared")
}

// GetMessageCount returns the number of messages in the conversation
func (a *SagaWorkflowAgent) GetMessageCount() int {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.messages)
}
