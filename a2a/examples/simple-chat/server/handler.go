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

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/types"
)

// EchoMessageHandler implements a simple echo chat handler that responds to messages
type EchoMessageHandler struct{}

// HandleMessage processes incoming messages and provides intelligent responses
func (h *EchoMessageHandler) HandleMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error) {
	// Get message content from first text part
	messageContent := h.getMessageContent(req.Message)
	log.Printf("Processing message: %s", messageContent)

	// Update task state to working
	err := req.TaskOperations.UpdateTaskState(types.TaskStateWorking, &types.Message{
		Role: types.RoleAgent,
		Parts: []types.Part{
			&types.TextPart{Text: "Processing your message..."},
		},
	})
	if err != nil {
		log.Printf("Failed to update task state: %v", err)
	}

	// Simulate some processing time
	time.Sleep(500 * time.Millisecond)

	// Analyze the message content and generate appropriate response
	response := h.generateResponse(messageContent)

	// Create response message
	responseMsg := &types.Message{
		Role: types.RoleAgent,
		Parts: []types.Part{
			&types.TextPart{Text: response},
		},
	}

	// Create artifact if the user asks for code
	if strings.Contains(strings.ToLower(messageContent), "code") ||
		strings.Contains(strings.ToLower(messageContent), "program") {
		artifact := &types.Artifact{
			ArtifactID:  "example-code",
			Name:        "Example Code",
			Description: "Simple Python example",
			Parts: []types.Part{
				&types.TextPart{Text: "print('Hello from A2A!')\nfor i in range(3):\n    print(f'Count: {i}')"},
			},
		}

		err := req.TaskOperations.AddArtifact(artifact)
		if err != nil {
			log.Printf("Failed to add artifact: %v", err)
		}
	}

	// Update task state to completed
	err = req.TaskOperations.UpdateTaskState(types.TaskStateCompleted, responseMsg)
	if err != nil {
		log.Printf("Failed to complete task: %v", err)
	}

	// Get the updated task after processing
	history, err := req.TaskOperations.GetTaskHistory()
	if err != nil {
		log.Printf("Failed to get task history: %v", err)
		history = []*types.Message{req.Message, responseMsg} // Fallback to basic history
	}

	// Create updated task object to return
	updatedTask := &types.Task{
		ID:        req.TaskID,
		ContextID: req.ContextID,
		Status: &types.TaskStatus{
			State:  types.TaskStateCompleted,
			Update: responseMsg,
		},
		History:   history,
		Artifacts: []*types.Artifact{}, // Will be populated if artifacts were added
		Kind:      "task",
		Metadata:  req.Metadata,
	}

	// Add artifacts if any were created
	if strings.Contains(strings.ToLower(messageContent), "code") ||
		strings.Contains(strings.ToLower(messageContent), "program") {
		// The artifact should already be added via TaskOperations.AddArtifact
		// But we need to include it in our response
		artifact := &types.Artifact{
			ArtifactID:  "example-code",
			Name:        "Example Code",
			Description: "Simple Python example",
			Parts: []types.Part{
				&types.TextPart{Text: "print('Hello from A2A!')\\nfor i in range(3):\\n    print(f'Count: {i}')"},
			},
		}
		updatedTask.Artifacts = []*types.Artifact{artifact}
	}

	// Return response with updated task
	return &handler.MessageResponse{
		Mode: handler.ResponseModeTask,
		Result: &handler.ResponseResult{
			Data: updatedTask,
		},
	}, nil
}

// HandleCancel handles task cancellation
func (h *EchoMessageHandler) HandleCancel(ctx context.Context, taskID string) error {
	log.Printf("Cancelling task: %s", taskID)
	return nil
}

// getMessageContent extracts text content from message parts
func (h *EchoMessageHandler) getMessageContent(msg *types.Message) string {
	if msg == nil || len(msg.Parts) == 0 {
		return ""
	}

	for _, part := range msg.Parts {
		if textPart, ok := part.(*types.TextPart); ok {
			return textPart.Text
		}
	}
	return ""
}

// generateResponse creates contextual responses based on message content
func (h *EchoMessageHandler) generateResponse(content string) string {
	content = strings.ToLower(content)

	switch {
	case strings.Contains(content, "hello") || strings.Contains(content, "hi"):
		return "Hello! I'm your friendly A2A assistant. How can I help you today?"

	case strings.Contains(content, "how are you") || strings.Contains(content, "how do you do"):
		return "I'm doing great! I'm an A2A agent ready to help you with various tasks."

	case strings.Contains(content, "time") || strings.Contains(content, "date"):
		return fmt.Sprintf("The current time is: %s", time.Now().Format("2006-01-02 15:04:05"))

	case strings.Contains(content, "weather"):
		return "I don't have access to real weather data, but I can help you with other tasks!"

	case strings.Contains(content, "code") || strings.Contains(content, "program"):
		return "I can help with coding! I've created a simple Python example for you. Check the artifacts section for the code."

	case strings.Contains(content, "joke"):
		return "Why don't scientists trust atoms? Because they make up everything! 😄"

	case strings.Contains(content, "help") || strings.Contains(content, "what can you do"):
		return "I can help you with:\n- General conversation\n- Providing current time\n- Creating simple code examples\n- Telling jokes\n- And much more! Just ask me anything."

	case strings.Contains(content, "thank") || strings.Contains(content, "thanks"):
		return "You're very welcome! I'm happy to help. Feel free to ask me anything else!"

	default:
		return fmt.Sprintf("I received your message: \"%s\"\n\nI'm an echo assistant that can help with various tasks. Try asking me for help, the time, a joke, or some code!", content)
	}
}
