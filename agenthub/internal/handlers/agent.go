package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"agenthub/pkg/common"
	"agenthub/pkg/models"
	"agenthub/pkg/utils"
)

// AgentHandler handles agent-related HTTP requests following K8s patterns
type AgentHandler struct {
	agentService *services.AgentService
	logger       *utils.Logger
	writer       *common.HTTPResponseWriter
}

// AgentHandlerConfig holds agent handler configuration
type AgentHandlerConfig struct {
	AgentService *services.AgentService
	Logger       *utils.Logger
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(config AgentHandlerConfig) *AgentHandler {
	logger := config.Logger
	if logger == nil {
		logger = utils.WithField("component", "agent-handler")
	}

	return &AgentHandler{
		agentService: config.AgentService,
		logger:       logger,
		writer:       common.NewHTTPResponseWriter(),
	}
}

// RegisterAgent handles agent registration
func (h *AgentHandler) RegisterAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid method for register: %s", r.Method)
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Error decoding register request: %v", err)
		h.writer.WriteBadRequest(w, "Invalid request body")
		return
	}

	ctx := r.Context()
	response, err := h.agentService.RegisterAgent(ctx, &req)
	if err != nil {
		h.logger.Error("Error registering agent: %v", err)
		h.writer.WriteInternalServerError(w, "Internal server error")
		return
	}

	if !response.Success {
		h.writer.WriteJSONResponse(w, response, http.StatusBadRequest)
		return
	}

	h.logger.Info("Successfully registered agent: %s", response.AgentID)
	h.writer.WriteSuccessResponse(w, response, "Agent registered successfully")
}

// DiscoverAgents handles agent discovery
func (h *AgentHandler) DiscoverAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid method for discover: %s", r.Method)
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	var req models.DiscoverRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Error decoding discover request: %v", err)
		h.writer.WriteBadRequest(w, "Invalid request body")
		return
	}

	ctx := r.Context()
	response, err := h.agentService.DiscoverAgents(ctx, &req)
	if err != nil {
		h.logger.Error("Error discovering agents: %v", err)
		h.writer.WriteInternalServerError(w, "Internal server error")
		return
	}

	h.logger.Info("Successfully discovered %d agents", len(response.Agents))
	h.writer.WriteSuccessResponse(w, response, "Agents discovered successfully")
}

// GetAgent handles getting a specific agent
func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	// Extract agent ID from URL path
	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		h.writer.WriteBadRequest(w, "Agent ID is required")
		return
	}

	ctx := r.Context()
	agent, err := h.agentService.GetAgent(ctx, agentID)
	if err != nil {
		h.logger.Error("Error getting agent %s: %v", agentID, err)
		h.writer.WriteInternalServerError(w, "Failed to get agent")
		return
	}

	h.writer.WriteSuccessResponse(w, agent, "Agent retrieved successfully")
}

// ListAgents handles listing all agents
func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	ctx := r.Context()
	agents, err := h.agentService.ListAllAgents(ctx)
	if err != nil {
		h.logger.Error("Error listing agents: %v", err)
		h.writer.WriteInternalServerError(w, "Failed to list agents")
		return
	}

	h.logger.Info("Successfully listed %d agents", len(agents))
	h.writer.WriteSuccessResponse(w, agents, "Agents listed successfully")
}

// UpdateAgentStatus handles updating agent status
func (h *AgentHandler) UpdateAgentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		h.writer.WriteBadRequest(w, "Agent ID is required")
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writer.WriteBadRequest(w, "Invalid request body")
		return
	}

	ctx := r.Context()
	if err := h.agentService.UpdateAgentStatus(ctx, agentID, req.Status); err != nil {
		h.logger.Error("Error updating agent status: %v", err)
		h.writer.WriteInternalServerError(w, "Failed to update agent status")
		return
	}

	h.logger.Info("Successfully updated agent %s status to %s", agentID, req.Status)
	h.writer.WriteSuccessResponse(w, nil, "Agent status updated successfully")
}

// RemoveAgent handles agent removal
func (h *AgentHandler) RemoveAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		h.writer.WriteBadRequest(w, "Agent ID is required")
		return
	}

	ctx := r.Context()
	if err := h.agentService.RemoveAgent(ctx, agentID); err != nil {
		h.logger.Error("Error removing agent %s: %v", agentID, err)
		h.writer.WriteInternalServerError(w, "Failed to remove agent")
		return
	}

	h.logger.Info("Successfully removed agent: %s", agentID)
	h.writer.WriteSuccessResponse(w, nil, "Agent removed successfully")
}

// Heartbeat handles agent heartbeat updates
func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		h.writer.WriteBadRequest(w, "Agent ID is required")
		return
	}

	ctx := r.Context()
	if err := h.agentService.UpdateAgentHeartbeat(ctx, agentID); err != nil {
		h.logger.Error("Error updating agent heartbeat: %v", err)
		h.writer.WriteInternalServerError(w, "Failed to update heartbeat")
		return
	}

	h.logger.Debug("Updated heartbeat for agent: %s", agentID)
	h.writer.WriteSuccessResponse(w, nil, "Heartbeat updated successfully")
}

// HealthCheck handles health check endpoint
func (h *AgentHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	status := map[string]interface{}{
		"status":    "healthy",
		"component": "agent-handler",
	}

	h.writer.WriteHealthResponse(w, status)
}

// AnalyzeContext handles dynamic context analysis requests
func (h *AgentHandler) AnalyzeContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid method for context analysis: %s", r.Method)
		h.writer.WriteMethodNotAllowed(w)
		return
	}

	var req models.ContextAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body: %v", err)
		h.writer.WriteBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.NeedDescription == "" {
		h.logger.Warn("Empty need description in context analysis request")
		h.writer.WriteBadRequest(w, "Need description is required")
		return
	}

	h.logger.Info("Processing context analysis for: %s", req.NeedDescription)

	ctx := r.Context()
	response, err := h.agentService.AnalyzeContext(ctx, &req)
	if err != nil {
		h.logger.Error("Context analysis failed: %v", err)
		h.writer.WriteErrorResponse(w, "Context analysis failed", http.StatusInternalServerError)
		return
	}

	h.logger.Info("Context analysis completed successfully")
	h.writer.WriteSuccessResponse(w, response, "Context analysis completed")
}

// AuthenticatedHandler wraps handlers that require authentication
func (h *AgentHandler) AuthenticatedHandler(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := common.GetClaimsFromContext(r.Context())
		if claims == nil {
			h.logger.Warn("Unauthenticated request to protected endpoint")
			h.writer.WriteUnauthorized(w, "Authentication required")
			return
		}

		// Add user info to logger context
		logger := h.logger.WithFields(map[string]interface{}{
			"user_id":  claims.UserID,
			"username": claims.Username,
		})

		// Create new handler context with enhanced logger
		ctx := context.WithValue(r.Context(), "logger", logger)
		handler(w, r.WithContext(ctx))
	}
}

// RequireRole wraps handlers that require specific roles
func (h *AgentHandler) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return h.AuthenticatedHandler(func(w http.ResponseWriter, r *http.Request) {
			claims := common.GetClaimsFromContext(r.Context())
			if !claims.HasAnyRole(roles...) {
				h.logger.Warn("Insufficient permissions for user %s", claims.Username)
				h.writer.WriteForbidden(w, "Insufficient permissions")
				return
			}

			handler(w, r)
		})
	}
}
