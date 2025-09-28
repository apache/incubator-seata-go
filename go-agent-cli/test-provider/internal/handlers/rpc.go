package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"test-provider/internal/config"
)

type AgentCardHandler struct {
	cfg *config.Config
}

func NewAgentCardHandler(cfg *config.Config) *AgentCardHandler {
	return &AgentCardHandler{cfg: cfg}
}

func (h *AgentCardHandler) Handle(c *gin.Context) {
	agentCard, err := h.cfg.LoadAgentCard()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load agent card"})
		return
	}

	c.JSON(http.StatusOK, agentCard)
}
