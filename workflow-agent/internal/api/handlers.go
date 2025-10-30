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

package api

import (
	"encoding/json"
	"net/http"

	"seata-go-ai-workflow-agent/internal/orchestration"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// Handler handles HTTP requests for workflow orchestration
type Handler struct {
	orchestrator *orchestration.Orchestrator
}

// NewHandler creates a new API handler
func NewHandler(orchestrator *orchestration.Orchestrator) *Handler {
	return &Handler{
		orchestrator: orchestrator,
	}
}

// OrchestrationRequest represents the API request for orchestration
type OrchestrationRequest struct {
	Description string                 `json:"description"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// OrchestrationResponse represents the API response
type OrchestrationResponse struct {
	Success      bool                                 `json:"success"`
	Message      string                               `json:"message"`
	RetryCount   int                                  `json:"retry_count"`
	Capabilities []orchestration.DiscoveredCapability `json:"capabilities,omitempty"`
	Workflow     *WorkflowResult                      `json:"workflow,omitempty"`
	Error        string                               `json:"error,omitempty"`
}

// WorkflowResult contains the generated workflow artifacts
type WorkflowResult struct {
	SeataWorkflow *orchestration.SeataStateMachine `json:"seata_workflow"`
	ReactFlow     *orchestration.ReactFlowGraph    `json:"react_flow"`
}

// HandleOrchestrate handles the orchestration endpoint
func (h *Handler) HandleOrchestrate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req OrchestrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.WithField("error", err).Error("failed to decode request")
		respondJSON(w, http.StatusBadRequest, OrchestrationResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	if req.Description == "" {
		respondJSON(w, http.StatusBadRequest, OrchestrationResponse{
			Success: false,
			Message: "Description is required",
			Error:   "description field cannot be empty",
		})
		return
	}

	logger.WithField("description", req.Description).Info("orchestration request received")

	// Perform orchestration
	scenarioReq := orchestration.ScenarioRequest{
		Description: req.Description,
		Context:     req.Context,
	}

	result, err := h.orchestrator.Orchestrate(r.Context(), scenarioReq)

	var resp OrchestrationResponse
	if err != nil {
		logger.WithField("error", err).Error("orchestration failed")
		resp = OrchestrationResponse{
			Success:      false,
			Message:      result.Message,
			RetryCount:   result.RetryCount,
			Capabilities: result.Capabilities,
			Error:        err.Error(),
		}
		respondJSON(w, http.StatusOK, resp)
		return
	}

	resp = OrchestrationResponse{
		Success:      result.Success,
		Message:      result.Message,
		RetryCount:   result.RetryCount,
		Capabilities: result.Capabilities,
	}

	if result.SeataWorkflow != nil && result.ReactFlow != nil {
		resp.Workflow = &WorkflowResult{
			SeataWorkflow: result.SeataWorkflow,
			ReactFlow:     result.ReactFlow,
		}
	}

	respondJSON(w, http.StatusOK, resp)
}

// HandleHealth handles the health check endpoint
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status":    "healthy",
		"component": "workflow-agent",
	})
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.WithField("error", err).Error("failed to encode response")
	}
}
