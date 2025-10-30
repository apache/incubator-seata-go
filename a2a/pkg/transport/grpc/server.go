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

package grpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"seata-go-ai-a2a/pkg/handler"
	"seata-go-ai-a2a/pkg/task"
	"seata-go-ai-a2a/pkg/types"

	pb "seata-go-ai-a2a/pkg/proto/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Server provides gRPC server implementation for A2A protocol
type Server struct {
	pb.UnimplementedA2AServiceServer

	grpcServer  *grpc.Server
	listener    net.Listener
	agentCard   *types.AgentCard
	taskManager *task.TaskManager

	// Push notification config storage (in-memory for now)
	pushNotificationConfigs map[string]*types.TaskPushNotificationConfig
	pushConfigMu            sync.RWMutex

	// Server lifecycle management
	mu      sync.RWMutex
	started bool
	stopped bool
}

// ServerConfig configures the gRPC server
type ServerConfig struct {
	Address     string
	AgentCard   *types.AgentCard
	TaskManager *task.TaskManager
	GRPCOptions []grpc.ServerOption
}

// NewServer creates a new gRPC server
func NewServer(config *ServerConfig) (*Server, error) {
	if config.AgentCard == nil {
		return nil, fmt.Errorf("agent card is required")
	}
	if config.TaskManager == nil {
		return nil, fmt.Errorf("task manager is required")
	}

	// Create listener
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", config.Address, err)
	}

	// Create gRPC server with default options
	grpcOptions := config.GRPCOptions
	if grpcOptions == nil {
		grpcOptions = []grpc.ServerOption{
			grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB
			grpc.MaxSendMsgSize(4 * 1024 * 1024), // 4MB
		}
	}

	grpcServer := grpc.NewServer(grpcOptions...)

	server := &Server{
		grpcServer:              grpcServer,
		listener:                listener,
		agentCard:               config.AgentCard,
		taskManager:             config.TaskManager,
		pushNotificationConfigs: make(map[string]*types.TaskPushNotificationConfig),
	}

	// Register A2A service
	pb.RegisterA2AServiceServer(grpcServer, server)

	return server, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("server already started")
	}
	if s.stopped {
		return fmt.Errorf("server has been stopped")
	}

	s.started = true

	// Start serving in a goroutine
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			// Log error but don't panic
			fmt.Printf("gRPC server error: %v\n", err)
		}
	}()

	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started || s.stopped {
		return nil
	}

	s.stopped = true
	s.grpcServer.GracefulStop()
	return nil
}

// GetAddress returns the server's listening address
func (s *Server) GetAddress() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return ""
}

// A2A Protocol Methods Implementation

// SendMessage sends a message to the agent
func (s *Server) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	// Convert protobuf request to types
	sendReq, err := types.SendMessageRequestFromProto(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request: %v", err))
	}

	// Check if task manager has handler interfaces
	if handlerTaskManager, ok := interface{}(s.taskManager).(interface {
		ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
	}); ok {
		// Use handler interface for processing
		return s.sendMessageWithHandler(ctx, sendReq, handlerTaskManager)
	}

	// Fallback to legacy direct task creation
	task, err := s.taskManager.CreateTask(ctx, sendReq.Request.ContextID, sendReq.Request, sendReq.Metadata)
	if err != nil {
		// Map task manager errors to gRPC status codes
		if isA2AError(err) {
			return nil, mapA2AErrorToGRPCStatus(err)
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create task: %v", err))
	}

	// Convert task to protobuf response
	pbTask, err := types.TaskToProto(task)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
	}

	return &pb.SendMessageResponse{
		Payload: &pb.SendMessageResponse_Task{Task: pbTask},
	}, nil
}

// SendStreamingMessage sends a message and streams task updates
func (s *Server) SendStreamingMessage(req *pb.SendMessageRequest, stream pb.A2AService_SendStreamingMessageServer) error {
	ctx := stream.Context()

	// Convert protobuf request to types
	sendReq, err := types.SendMessageRequestFromProto(req)
	if err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid request: %v", err))
	}

	// Create task using task manager
	task, err := s.taskManager.CreateTask(ctx, sendReq.Request.ContextID, sendReq.Request, sendReq.Metadata)
	if err != nil {
		if isA2AError(err) {
			return mapA2AErrorToGRPCStatus(err)
		}
		return status.Error(codes.Internal, fmt.Sprintf("failed to create task: %v", err))
	}

	// Subscribe to task updates
	subscription, err := s.taskManager.SubscribeToTask(ctx, task.ID)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to subscribe to task: %v", err))
	}
	defer subscription.Close()

	// Send initial task state
	pbTask, err := types.TaskToProto(task)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
	}

	initialResponse := &pb.StreamResponse{
		Payload: &pb.StreamResponse_Task{Task: pbTask},
	}

	if err := stream.Send(initialResponse); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to send initial response: %v", err))
	}

	// Stream task updates
	for {
		select {
		case update, ok := <-subscription.Events():
			if !ok {
				// Subscription closed
				return nil
			}

			// Convert update to protobuf stream response
			pbResponse, err := convertStreamResponseToProto(update)
			if err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("failed to convert stream response: %v", err))
			}

			if err := stream.Send(pbResponse); err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("failed to send stream response: %v", err))
			}

		case <-ctx.Done():
			return status.Error(codes.Canceled, "stream cancelled by client")
		}
	}
}

// GetTask retrieves a task by ID
func (s *Server) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.Task, error) {
	// Convert protobuf request to types
	getReq := types.GetTaskRequestFromProto(req)

	// Extract task ID from name (format: tasks/{task_id})
	taskID := extractTaskIDFromName(getReq.Name)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid task name format")
	}

	// Get task from task manager
	task, err := s.taskManager.GetTask(ctx, taskID)
	if err != nil {
		if isA2AError(err) {
			return nil, mapA2AErrorToGRPCStatus(err)
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get task: %v", err))
	}

	// Apply history length limit if specified
	if req.HistoryLength > 0 && int(req.HistoryLength) < len(task.History) {
		// Create a copy with limited history
		taskCopy := *task
		taskCopy.History = task.History[:req.HistoryLength]
		task = &taskCopy
	}

	// Convert to protobuf
	pbTask, err := types.TaskToProto(task)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
	}

	return pbTask, nil
}

// CancelTask cancels a task
func (s *Server) CancelTask(ctx context.Context, req *pb.CancelTaskRequest) (*pb.Task, error) {
	// Convert protobuf request to types
	cancelReq := types.CancelTaskRequestFromProto(req)

	// Extract task ID from name
	taskID := extractTaskIDFromName(cancelReq.Name)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid task name format")
	}

	// Cancel task using task manager
	task, err := s.taskManager.CancelTask(ctx, taskID)
	if err != nil {
		if isA2AError(err) {
			return nil, mapA2AErrorToGRPCStatus(err)
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to cancel task: %v", err))
	}

	// Convert to protobuf
	pbTask, err := types.TaskToProto(task)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
	}

	return pbTask, nil
}

// TaskSubscription subscribes to task updates
func (s *Server) TaskSubscription(req *pb.TaskSubscriptionRequest, stream pb.A2AService_TaskSubscriptionServer) error {
	ctx := stream.Context()

	// Convert protobuf request to types
	subReq := types.TaskSubscriptionRequestFromProto(req)

	// Extract task ID from name
	taskID := extractTaskIDFromName(subReq.Name)
	if taskID == "" {
		return status.Error(codes.InvalidArgument, "invalid task name format")
	}

	// Subscribe to task updates
	subscription, err := s.taskManager.SubscribeToTask(ctx, taskID)
	if err != nil {
		if isA2AError(err) {
			return mapA2AErrorToGRPCStatus(err)
		}
		return status.Error(codes.Internal, fmt.Sprintf("failed to subscribe to task: %v", err))
	}
	defer subscription.Close()

	// Stream task updates
	for {
		select {
		case update, ok := <-subscription.Events():
			if !ok {
				// Subscription closed
				return nil
			}

			// Convert update to protobuf stream response
			pbResponse, err := convertStreamResponseToProto(update)
			if err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("failed to convert stream response: %v", err))
			}

			if err := stream.Send(pbResponse); err != nil {
				return status.Error(codes.Internal, fmt.Sprintf("failed to send stream response: %v", err))
			}

		case <-ctx.Done():
			return status.Error(codes.Canceled, "stream cancelled by client")
		}
	}
}

// ListTasks lists tasks with optional filtering
func (s *Server) ListTasks(ctx context.Context, req *pb.ListTasksRequest) (*pb.ListTasksResponse, error) {
	// Convert protobuf request to types
	listReq := types.ListTasksRequestFromProto(req)

	// List tasks using task manager
	response, err := s.taskManager.ListTasks(ctx, listReq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list tasks: %v", err))
	}

	// Convert to protobuf
	pbResponse, err := types.ListTasksResponseToProto(response)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert response: %v", err))
	}

	return pbResponse, nil
}

// GetAgentCard retrieves the agent card
func (s *Server) GetAgentCard(ctx context.Context, req *pb.GetAgentCardRequest) (*pb.AgentCard, error) {
	var agentCard *types.AgentCard

	// Check if task manager has agent card provider
	if cardProvider, ok := interface{}(s.taskManager).(interface {
		GetAgentCard(ctx context.Context) (*types.AgentCard, error)
	}); ok {
		// Use provider interface
		card, err := cardProvider.GetAgentCard(ctx)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to get agent card from provider: %v", err))
		}
		agentCard = card
	} else {
		// Fallback to static agent card
		agentCard = s.agentCard
	}

	if agentCard == nil {
		return nil, status.Error(codes.Internal, "no agent card available")
	}

	// Convert agent card to protobuf
	pbAgentCard, err := types.AgentCardToProto(agentCard)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert agent card: %v", err))
	}

	return pbAgentCard, nil
}

// CreateTaskPushNotification creates a push notification config
func (s *Server) CreateTaskPushNotification(ctx context.Context, req *pb.CreateTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	// Parse task ID from parent (format: tasks/{task_id})
	taskID := parseTaskIDFromResourceName(req.Parent)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid parent format, expected tasks/{task_id}")
	}

	// Verify task exists
	_, err := s.taskManager.GetTask(ctx, taskID)
	if err != nil {
		if isA2ATaskNotFoundError(err) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to verify task: %v", err))
	}

	// Convert config from proto
	config, err := types.TaskPushNotificationConfigFromProto(req.Config)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid config: %v", err))
	}

	// Generate config ID if not provided
	configID := req.ConfigId
	if configID == "" {
		configID = uuid.New().String()
	}

	// Generate full name for the config
	configName := fmt.Sprintf("tasks/%s/pushNotificationConfigs/%s", taskID, configID)
	config.Name = configName

	// Store the config
	s.pushConfigMu.Lock()
	s.pushNotificationConfigs[configName] = config
	s.pushConfigMu.Unlock()

	// Convert back to proto and return
	pbConfig, err := types.TaskPushNotificationConfigToProto(config)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert config: %v", err))
	}

	return pbConfig, nil
}

// GetTaskPushNotification retrieves a push notification config
func (s *Server) GetTaskPushNotification(ctx context.Context, req *pb.GetTaskPushNotificationConfigRequest) (*pb.TaskPushNotificationConfig, error) {
	// Get the config from storage
	s.pushConfigMu.RLock()
	config, exists := s.pushNotificationConfigs[req.Name]
	s.pushConfigMu.RUnlock()

	if !exists {
		return nil, status.Error(codes.NotFound, "push notification config not found")
	}

	// Convert to proto and return
	pbConfig, err := types.TaskPushNotificationConfigToProto(config)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert config: %v", err))
	}

	return pbConfig, nil
}

// ListTaskPushNotification lists push notification configs
func (s *Server) ListTaskPushNotification(ctx context.Context, req *pb.ListTaskPushNotificationConfigRequest) (*pb.ListTaskPushNotificationConfigResponse, error) {
	// Parse task ID from parent (format: tasks/{task_id})
	taskID := parseTaskIDFromResourceName(req.Parent)
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid parent format, expected tasks/{task_id}")
	}

	// Verify task exists
	_, err := s.taskManager.GetTask(ctx, taskID)
	if err != nil {
		if isA2ATaskNotFoundError(err) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to verify task: %v", err))
	}

	// Find all configs for this task
	s.pushConfigMu.RLock()
	var configs []*types.TaskPushNotificationConfig
	prefix := fmt.Sprintf("tasks/%s/pushNotificationConfigs/", taskID)

	for name, config := range s.pushNotificationConfigs {
		if strings.HasPrefix(name, prefix) {
			configs = append(configs, config)
		}
	}
	s.pushConfigMu.RUnlock()

	// Convert to proto response
	response := &types.ListTaskPushNotificationConfigResponse{
		Configs: configs,
	}

	pbResponse, err := types.ListTaskPushNotificationConfigResponseToProto(response)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert response: %v", err))
	}

	return pbResponse, nil
}

// DeleteTaskPushNotification deletes a push notification config
func (s *Server) DeleteTaskPushNotification(ctx context.Context, req *pb.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	// Check if config exists
	s.pushConfigMu.Lock()
	defer s.pushConfigMu.Unlock()

	_, exists := s.pushNotificationConfigs[req.Name]
	if !exists {
		return nil, status.Error(codes.NotFound, "push notification config not found")
	}

	// Delete the config
	delete(s.pushNotificationConfigs, req.Name)

	return &emptypb.Empty{}, nil
}

// UpdateAgentCard updates the agent card served by the gRPC server
func (s *Server) UpdateAgentCard(agentCard *types.AgentCard) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agentCard = agentCard
}

// GetCurrentAgentCard returns the current agent card
func (s *Server) GetCurrentAgentCard() *types.AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.agentCard
}

// SetTaskManager updates the task manager used by the server
func (s *Server) SetTaskManager(taskManager *task.TaskManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskManager = taskManager
}

// Validate validates the server configuration
func (s *Server) Validate() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.agentCard == nil {
		return fmt.Errorf("agent card is required")
	}
	if s.taskManager == nil {
		return fmt.Errorf("task manager is required")
	}

	return nil
}

// parseTaskIDFromResourceName parses task ID from resource name (e.g., "tasks/{task_id}")
func parseTaskIDFromResourceName(name string) string {
	const prefix = "tasks/"
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	// Find the next slash or take the rest if no slash
	remaining := name[len(prefix):]
	if idx := strings.Index(remaining, "/"); idx != -1 {
		return remaining[:idx]
	}
	return remaining
}

// isA2ATaskNotFoundError checks if an error is an A2A TaskNotFoundError
func isA2ATaskNotFoundError(err error) bool {
	if a2aErr, ok := err.(*types.A2AError); ok {
		return a2aErr.Code == types.TaskNotFoundError
	}
	return false
}

// sendMessageWithHandler processes message through handler interface
func (s *Server) sendMessageWithHandler(ctx context.Context, sendReq *types.SendMessageRequest, handlerTaskManager interface {
	ProcessMessage(ctx context.Context, req *handler.MessageRequest) (*handler.MessageResponse, error)
}) (*pb.SendMessageResponse, error) {
	// Create task first to get task ID
	task, err := s.taskManager.CreateTask(ctx, sendReq.Request.ContextID, sendReq.Request, sendReq.Metadata)
	if err != nil {
		// Map task manager errors to gRPC status codes
		if isA2AError(err) {
			return nil, mapA2AErrorToGRPCStatus(err)
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to create task: %v", err))
	}

	// Create task operations for handler
	eventChan := make(chan types.StreamResponse, 100)
	taskOps := handler.NewTaskOperations(s.taskManager, task.ID, sendReq.Request.ContextID, eventChan)

	// Build message request for handler
	messageReq := &handler.MessageRequest{
		Message:   sendReq.Request,
		TaskID:    task.ID,
		ContextID: sendReq.Request.ContextID,
		Configuration: &handler.MessageConfiguration{
			Blocking:  true, // gRPC is blocking by default
			Streaming: false,
			Timeout:   30 * time.Second,
		},
		History:        task.History,
		Metadata:       sendReq.Metadata,
		TaskOperations: taskOps,
	}

	// Process through handler
	response, err := handlerTaskManager.ProcessMessage(ctx, messageReq)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("handler processing failed: %v", err))
	}

	// Handle response based on mode
	switch response.Mode {
	case handler.ResponseModeTask:
		// Return task response
		if response.Result != nil && response.Result.Data != nil {
			if taskData, ok := response.Result.Data.(*types.Task); ok {
				pbTask, err := types.TaskToProto(taskData)
				if err != nil {
					return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
				}
				return &pb.SendMessageResponse{
					Payload: &pb.SendMessageResponse_Task{Task: pbTask},
				}, nil
			}
		}
	case handler.ResponseModeMessage:
		// Return message response
		if response.Result != nil && response.Result.Data != nil {
			if msgData, ok := response.Result.Data.(*types.Message); ok {
				pbMsg, err := types.MessageToProto(msgData)
				if err != nil {
					return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert message: %v", err))
				}
				return &pb.SendMessageResponse{
					Payload: &pb.SendMessageResponse_Msg{Msg: pbMsg},
				}, nil
			}
		}
	default:
		// For other modes or errors, return the original task
		pbTask, err := types.TaskToProto(task)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
		}
		return &pb.SendMessageResponse{
			Payload: &pb.SendMessageResponse_Task{Task: pbTask},
		}, nil
	}

	// Fallback - return original task
	pbTask, err := types.TaskToProto(task)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to convert task: %v", err))
	}
	return &pb.SendMessageResponse{
		Payload: &pb.SendMessageResponse_Task{Task: pbTask},
	}, nil
}
