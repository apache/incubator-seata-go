package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agenthub/internal/handlers"
	"agenthub/internal/services"
	"agenthub/pkg/auth"
	"agenthub/pkg/config"
	"agenthub/pkg/server"
	"agenthub/pkg/storage"
	"agenthub/pkg/utils"
)

// Application represents the main application following K8s patterns
type Application struct {
	config         *config.Config
	logger         *utils.Logger
	storage        storage.Storage
	agentService   *services.AgentService
	agentHandler   *handlers.AgentHandler
	jwtManager     *auth.JWTManager
	authMiddleware *auth.Middleware
	httpServer     *server.HTTPServer
}

// ApplicationConfig holds application configuration
type ApplicationConfig struct {
	ConfigPath string
}

// NewApplication creates a new application instance
func NewApplication(cfg ApplicationConfig) (*Application, error) {
	// Load configuration
	configLoader := config.NewConfigLoader()
	appConfig, err := configLoader.LoadFromFile(cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := appConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Debug: Print loaded configuration
	fmt.Printf("DEBUG: Loaded config - AI Provider: %s, Context Analysis Enabled: %v\n",
		appConfig.AI.Provider, appConfig.ContextAnalysis.Enabled)

	// Initialize logger
	logger := utils.NewLogger(utils.LoggerConfig{
		Level:  appConfig.Logging.Level,
		Prefix: "[AgentHub] ",
		Fields: map[string]interface{}{
			"version": appConfig.Hub.Version,
			"hub_id":  appConfig.Hub.ID,
		},
	})

	app := &Application{
		config: appConfig,
		logger: logger,
	}

	// Initialize components
	if err := app.initializeStorage(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := app.initializeAuth(); err != nil {
		return nil, fmt.Errorf("failed to initialize auth: %w", err)
	}

	if err := app.initializeHandlers(); err != nil {
		return nil, fmt.Errorf("failed to initialize handlers: %w", err)
	}

	if err := app.initializeServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}

	return app, nil
}

// initializeStorage initializes the storage layer
func (a *Application) initializeStorage() error {
	// Check if NamingServer is enabled
	if a.config.NamingServer.Enabled {
		a.logger.Info("Initializing skill-based NamingServer storage")

		// TODO: Create actual NamingServer registry connection
		// For now, we'll use a mock registry since the seata-go implementation is not complete
		// registry, err := seata.NewNamingServerRegistry(a.config.NamingServer.Address,
		//     a.config.NamingServer.Username, a.config.NamingServer.Password)
		// if err != nil {
		//     return fmt.Errorf("failed to connect to NamingServer: %w", err)
		// }

		// 使用Mock注册中心进行测试和开发
		mockRegistry := storage.NewMockNamingserverRegistry()

		a.storage = storage.NewSkillBasedNamingServerStorage(mockRegistry)
		a.logger.Info("Initialized skill-based NamingServer storage (using mock registry)")
		return nil
	}

	// Fallback to traditional storage types
	switch a.config.Storage.Type {
	case "memory":
		a.storage = storage.NewMemoryStorage()
		a.logger.Info("Initialized memory storage")
	default:
		return fmt.Errorf("unsupported storage type: %s", a.config.Storage.Type)
	}

	return nil
}

// initializeServices initializes business services
func (a *Application) initializeServices() error {
	// Initialize context analyzer if enabled
	var contextAnalyzer *services.ContextAnalyzer
	a.logger.Debug("Context analysis enabled: %v, AI provider: %s", a.config.ContextAnalysis.Enabled, a.config.AI.Provider)
	if a.config.ContextAnalysis.Enabled {
		// Parse AI timeout
		aiTimeout, err := time.ParseDuration(a.config.AI.Timeout)
		if err != nil {
			a.logger.Warn("Invalid AI timeout, using default: %v", err)
			aiTimeout = 30 * time.Second
		}

		// Create AI client based on configuration
		aiClient := services.NewAIClient(services.AIClientConfig{
			Provider:  a.config.AI.Provider,
			APIKey:    a.config.AI.APIKey,
			BaseURL:   a.config.AI.BaseURL,
			Model:     a.config.AI.Model,
			MaxTokens: a.config.AI.MaxTokens,
			Timeout:   aiTimeout,
		})

		// Create context analyzer
		contextAnalyzer = services.NewContextAnalyzer(services.ContextAnalyzerConfig{
			Storage:  a.storage,
			Logger:   a.logger.WithField("component", "context-analyzer"),
			AIClient: aiClient,
		})

		a.logger.Info("Initialized context analyzer with AI provider: %s", a.config.AI.Provider)
	} else {
		a.logger.Info("Context analysis disabled")
	}

	// Initialize agent service
	a.agentService = services.NewAgentService(services.AgentServiceConfig{
		Storage:         a.storage,
		Logger:          a.logger.WithField("component", "agent-service"),
		ContextAnalyzer: contextAnalyzer,
	})

	a.logger.Info("Initialized agent service")
	return nil
}

// initializeAuth initializes authentication components
func (a *Application) initializeAuth() error {
	if !a.config.Auth.Enabled {
		a.logger.Info("Authentication disabled")
		return nil
	}

	// Initialize JWT manager
	jwtConfig := auth.JWTConfig{
		SecretKey: a.config.Auth.JWTSecret,
		Issuer:    a.config.Hub.Name,
		Expiry:    a.config.GetJWTExpiry(),
	}
	a.jwtManager = auth.NewJWTManager(jwtConfig)

	// Initialize auth middleware
	middlewareConfig := auth.MiddlewareConfig{
		JWTManager: a.jwtManager,
		Optional:   a.config.Auth.Optional,
		Logger:     a.logger.WithField("component", "auth-middleware"),
	}
	a.authMiddleware = auth.NewMiddleware(middlewareConfig)

	a.logger.Info("Initialized JWT authentication (optional: %v, expiry: %v)",
		a.config.Auth.Optional, a.config.GetJWTExpiry())
	return nil
}

// initializeHandlers initializes HTTP handlers
func (a *Application) initializeHandlers() error {
	a.agentHandler = handlers.NewAgentHandler(handlers.AgentHandlerConfig{
		AgentService: a.agentService,
		Logger:       a.logger.WithField("component", "agent-handler"),
	})

	a.logger.Info("Initialized HTTP handlers")
	return nil
}

// initializeServer initializes the HTTP server
func (a *Application) initializeServer() error {
	builder := server.NewServerBuilder().
		WithAddress(a.config.Hub.ListenAddress).
		WithReadTimeout(30*time.Second).
		WithWriteTimeout(30*time.Second).
		WithLogger(a.logger.WithField("component", "http-server")).
		WithMiddleware(
			server.RecoveryMiddleware(a.logger),
			server.LoggingMiddleware(a.logger),
			server.CORSMiddleware(),
		)

	// Add authentication middleware if enabled
	if a.authMiddleware != nil {
		builder = builder.WithAuthMiddleware(a.authMiddleware)
	}

	// Add routes
	routes := []server.Route{
		{Method: "GET", Path: "/", Handler: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("AgentHub is running"))
		}},
		{Method: "POST", Path: "/agent/register", Handler: a.agentHandler.RegisterAgent},
		{Method: "POST", Path: "/agent/discover", Handler: a.agentHandler.DiscoverAgents},
		{Method: "GET", Path: "/agent/get", Handler: a.agentHandler.GetAgent},
		{Method: "GET", Path: "/agents", Handler: a.agentHandler.ListAgents},
		{Method: "PUT", Path: "/agent/status", Handler: a.agentHandler.UpdateAgentStatus},
		{Method: "DELETE", Path: "/agent/remove", Handler: a.agentHandler.RemoveAgent},
		{Method: "POST", Path: "/agent/heartbeat", Handler: a.agentHandler.Heartbeat},
		{Method: "POST", Path: "/agent/analyze", Handler: a.agentHandler.AnalyzeContext},
		{Method: "GET", Path: "/health", Handler: a.agentHandler.HealthCheck},
	}

	// Add auth endpoints if JWT manager is available
	if a.jwtManager != nil {
		routes = append(routes,
			server.Route{Method: "POST", Path: "/auth/token", Handler: a.generateTokenHandler},
			server.Route{Method: "POST", Path: "/auth/refresh", Handler: a.refreshTokenHandler},
		)
	}

	builder = builder.WithRoutes(routes)
	a.httpServer = builder.Build()

	a.logger.Info("Initialized HTTP server")
	return nil
}

// Run starts the application
func (a *Application) Run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health check routine
	go a.startHealthCheckRoutine(ctx)

	// Start metrics server if enabled
	if a.config.Metrics.Enabled {
		go a.startMetricsServer()
	}

	// Setup graceful shutdown
	go a.setupGracefulShutdown(cancel)

	a.logger.Info("AgentHub starting...")
	a.logger.Info("Hub ID: %s, Version: %s", a.config.Hub.ID, a.config.Hub.Version)
	a.logger.Info("HTTP server listening on %s", a.config.Hub.ListenAddress)

	// Start HTTP server
	return a.httpServer.Start()
}

// startHealthCheckRoutine starts the health check routine
func (a *Application) startHealthCheckRoutine(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.agentService.HealthCheck(ctx, 5*time.Minute); err != nil {
				a.logger.Error("Health check failed: %v", err)
			}
		}
	}
}

// startMetricsServer starts the metrics server
func (a *Application) startMetricsServer() {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Simple metrics endpoint - can be extended with Prometheus metrics
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	a.logger.Info("Metrics server starting on %s", a.config.Metrics.ListenAddress)
	if err := http.ListenAndServe(a.config.Metrics.ListenAddress, nil); err != nil {
		a.logger.Error("Metrics server error: %v", err)
	}
}

// setupGracefulShutdown sets up graceful shutdown handling
func (a *Application) setupGracefulShutdown(cancel context.CancelFunc) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	a.logger.Info("Received shutdown signal, shutting down gracefully...")
	cancel()

	// Close storage
	if a.storage != nil {
		if err := a.storage.Close(); err != nil {
			a.logger.Error("Error closing storage: %v", err)
		}
	}

	a.logger.Info("AgentHub shutdown complete")
	os.Exit(0)
}

// generateTokenHandler handles token generation
func (a *Application) generateTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string   `json:"username"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
	}

	if err := utils.UnmarshalJSONFromReader(r.Body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Simple demo authentication - replace with real authentication
	if utils.IsEmpty(req.Username) {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	// Default roles
	if len(req.Roles) == 0 {
		req.Roles = []string{"user"}
	}

	token, err := a.jwtManager.GenerateToken("user-"+req.Username, req.Username, req.Roles)
	if err != nil {
		a.logger.Error("Failed to generate token: %v", err)
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"token":      token,
		"expires_in": a.config.Auth.JWTExpiry,
		"token_type": "Bearer",
	}

	w.Header().Set("Content-Type", "application/json")
	utils.MarshalJSONToWriter(w, response)
}

// refreshTokenHandler handles token refresh
func (a *Application) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	if err := utils.UnmarshalJSONFromReader(r.Body, &req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	newToken, err := a.jwtManager.RefreshToken(req.Token)
	if err != nil {
		a.logger.Error("Failed to refresh token: %v", err)
		http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"token":      newToken,
		"expires_in": a.config.Auth.JWTExpiry,
		"token_type": "Bearer",
	}

	w.Header().Set("Content-Type", "application/json")
	utils.MarshalJSONToWriter(w, response)
}
