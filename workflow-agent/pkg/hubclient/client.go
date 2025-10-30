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

package hubclient

import (
	"context"
	"time"
)

import (
	"agenthub/pkg/client"
)

import (
	"seata-go-ai-workflow-agent/pkg/errors"
	"seata-go-ai-workflow-agent/pkg/logger"
)

// HubClient wraps the AgentHub client with additional functionality
type HubClient struct {
	client  *client.Client
	agentID string
	config  *Config
}

// Config holds the configuration for HubClient
type Config struct {
	BaseURL          string
	Timeout          time.Duration
	AuthToken        string
	HeartbeatEnabled bool
	HeartbeatPeriod  time.Duration
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:          "http://localhost:8080",
		Timeout:          30 * time.Second,
		HeartbeatEnabled: true,
		HeartbeatPeriod:  30 * time.Second,
	}
}

// New creates a new HubClient instance
func New(cfg *Config) *HubClient {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	clientCfg := client.ClientConfig{
		BaseURL:   cfg.BaseURL,
		Timeout:   cfg.Timeout,
		AuthToken: cfg.AuthToken,
	}

	return &HubClient{
		client: client.NewClient(clientCfg),
		config: cfg,
	}
}

// HealthCheck performs a health check on the AgentHub
func (hc *HubClient) HealthCheck(ctx context.Context) error {
	logger.Debug("performing health check")

	resp, err := hc.client.HealthCheck(ctx)
	if err != nil {
		logger.WithField("error", err).Error("health check failed")
		return errors.Wrap(errors.ErrConnection, "health check failed", err)
	}

	logger.WithFields(map[string]interface{}{
		"component": resp.Component,
		"status":    resp.Status,
	}).Info("health check successful")

	return nil
}

// Register registers an agent with the AgentHub
func (hc *HubClient) Register(ctx context.Context, req *client.RegisterRequest) (string, error) {
	logger.WithFields(map[string]interface{}{
		"name": req.AgentCard.Name,
		"host": req.Host,
		"port": req.Port,
	}).Info("registering agent")

	resp, err := hc.client.RegisterAgent(ctx, req)
	if err != nil {
		logger.WithField("error", err).Error("failed to register agent")
		return "", errors.Wrap(errors.ErrRegistration, "registration failed", err)
	}

	if !resp.Success {
		logger.WithField("message", resp.Message).Warn("registration unsuccessful")
		return "", errors.RegistrationErrorf("registration unsuccessful: %s", resp.Message)
	}

	hc.agentID = resp.AgentID
	logger.WithField("agentID", hc.agentID).Info("agent registered successfully")

	if hc.config.HeartbeatEnabled {
		go hc.startHeartbeat(ctx)
	}

	return hc.agentID, nil
}

// Discover discovers agents based on a query
func (hc *HubClient) Discover(ctx context.Context, query string) ([]client.AgentCard, error) {
	logger.WithField("query", query).Debug("discovering agents")

	req := &client.DiscoverRequest{Query: query}
	resp, err := hc.client.DiscoverAgents(ctx, req)
	if err != nil {
		logger.WithField("error", err).Error("discovery failed")
		return nil, errors.Wrap(errors.ErrDiscovery, "discovery failed", err)
	}

	logger.WithField("count", len(resp.Agents)).Info("agents discovered")
	return resp.Agents, nil
}

// GetAgent retrieves information about a specific agent
func (hc *HubClient) GetAgent(ctx context.Context, agentID string) (*client.RegisteredAgent, error) {
	logger.WithField("agentID", agentID).Debug("getting agent info")

	agent, err := hc.client.GetAgent(ctx, agentID)
	if err != nil {
		logger.WithField("error", err).Error("failed to get agent")
		return nil, errors.Wrap(errors.ErrNotFound, "failed to get agent", err)
	}

	return agent, nil
}

// ListAgents lists all registered agents
func (hc *HubClient) ListAgents(ctx context.Context) ([]*client.RegisteredAgent, error) {
	logger.Debug("listing all agents")

	agents, err := hc.client.ListAgents(ctx)
	if err != nil {
		logger.WithField("error", err).Error("failed to list agents")
		return nil, errors.Wrap(errors.ErrDiscovery, "failed to list agents", err)
	}

	logger.WithField("count", len(agents)).Info("agents listed")
	return agents, nil
}

// UpdateStatus updates the agent's status
func (hc *HubClient) UpdateStatus(ctx context.Context, status string) error {
	if hc.agentID == "" {
		return errors.InvalidArgumentError("agent not registered")
	}

	logger.WithFields(map[string]interface{}{
		"agentID": hc.agentID,
		"status":  status,
	}).Debug("updating agent status")

	err := hc.client.UpdateAgentStatus(ctx, hc.agentID, status)
	if err != nil {
		logger.WithField("error", err).Error("failed to update status")
		return errors.Wrap(errors.ErrConnection, "failed to update status", err)
	}

	logger.Info("agent status updated")
	return nil
}

// Unregister removes the agent from the AgentHub
func (hc *HubClient) Unregister(ctx context.Context) error {
	if hc.agentID == "" {
		return errors.InvalidArgumentError("agent not registered")
	}

	logger.WithField("agentID", hc.agentID).Info("unregistering agent")

	err := hc.client.RemoveAgent(ctx, hc.agentID)
	if err != nil {
		logger.WithField("error", err).Error("failed to unregister agent")
		return errors.Wrap(errors.ErrRegistration, "failed to unregister", err)
	}

	logger.Info("agent unregistered successfully")
	hc.agentID = ""
	return nil
}

// AnalyzeContext performs dynamic context analysis
func (hc *HubClient) AnalyzeContext(ctx context.Context, needDescription, userContext string) (*client.ContextAnalysisResponse, error) {
	logger.WithField("needDescription", needDescription).Debug("analyzing context")

	req := &client.ContextAnalysisRequest{
		NeedDescription: needDescription,
		UserContext:     userContext,
	}

	resp, err := hc.client.AnalyzeContext(ctx, req)
	if err != nil {
		logger.WithField("error", err).Error("context analysis failed")
		return nil, errors.Wrap(errors.ErrConnection, "context analysis failed", err)
	}

	logger.WithField("success", resp.Success).Info("context analysis completed")
	return resp, nil
}

// GetAgentID returns the current agent ID
func (hc *HubClient) GetAgentID() string {
	return hc.agentID
}

// startHeartbeat starts sending periodic heartbeats
func (hc *HubClient) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(hc.config.HeartbeatPeriod)
	defer ticker.Stop()

	logger.WithField("period", hc.config.HeartbeatPeriod).Info("heartbeat started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("heartbeat stopped: context cancelled")
			return
		case <-ticker.C:
			if hc.agentID == "" {
				logger.Warn("heartbeat skipped: agent not registered")
				continue
			}

			if err := hc.sendHeartbeat(ctx); err != nil {
				logger.WithField("error", err).Error("heartbeat failed")
			}
		}
	}
}

// sendHeartbeat sends a single heartbeat
func (hc *HubClient) sendHeartbeat(ctx context.Context) error {
	logger.WithField("agentID", hc.agentID).Debug("sending heartbeat")

	err := hc.client.SendHeartbeat(ctx, hc.agentID)
	if err != nil {
		return errors.Wrap(errors.ErrHeartbeat, "heartbeat failed", err)
	}

	logger.Debug("heartbeat sent successfully")
	return nil
}
