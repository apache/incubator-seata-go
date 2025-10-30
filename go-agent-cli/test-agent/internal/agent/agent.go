package agent

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"test-agent/internal/agent/handlers"
	"test-agent/internal/agent/skills"
	"test-agent/internal/config"
)

type Agent struct {
	cfg    *config.Config
	logger *zap.Logger
	router *gin.Engine
	skills map[string]skills.Skill
}

func New(cfg *config.Config, logger *zap.Logger) *Agent {
	a := &Agent{
		cfg:    cfg,
		logger: logger,
		router: gin.New(),
		skills: make(map[string]skills.Skill),
	}

	a.initializeSkills()
	a.setupRoutes()
	return a
}

func (a *Agent) initializeSkills() {
	a.skills["example"] = skills.NewExampleSkill(a.logger)
}

func (a *Agent) setupRoutes() {
	a.router.Use(gin.Logger())
	a.router.Use(gin.Recovery())

	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	a.router.GET("/agent-card", handlers.NewAgentCardHandler(a.cfg).Handle)

	skillsGroup := a.router.Group("/skills")
	for name, skill := range a.skills {
		skillsGroup.POST("/"+name, a.createSkillHandler(skill))
	}
}

func (a *Agent) createSkillHandler(skill skills.Skill) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request map[string]interface{}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := skill.Execute(context.Background(), request)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

func (a *Agent) Start() error {
	addr := fmt.Sprintf("%s:%d", a.cfg.Server.Host, a.cfg.Server.Port)
	a.logger.Info("Starting agent server", zap.String("address", addr))
	return a.router.Run(addr)
}
