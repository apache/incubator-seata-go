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

package types

import (
	"fmt"

	pb "seata-go-ai-a2a/pkg/proto/v1"
)

// SendMessageRequest represents a request to send a message to an agent
type SendMessageRequest struct {
	Request       *Message                  `json:"message"`
	Configuration *SendMessageConfiguration `json:"configuration,omitempty"`
	Metadata      map[string]any            `json:"metadata,omitempty"`
}

// GetTaskRequest represents a request to get task information
type GetTaskRequest struct {
	Name          string `json:"name"` // Format: tasks/{task_id}
	HistoryLength int32  `json:"history_length,omitempty"`
}

// CancelTaskRequest represents a request to cancel a task
type CancelTaskRequest struct {
	Name string `json:"name"` // Format: tasks/{task_id}
}

// GetTaskPushNotificationConfigRequest represents a request to get push notification config
type GetTaskPushNotificationConfigRequest struct {
	Name string `json:"name"` // Format: tasks/{task_id}/pushNotificationConfigs/{config_id}
}

// DeleteTaskPushNotificationConfigRequest represents a request to delete push notification config
type DeleteTaskPushNotificationConfigRequest struct {
	Name string `json:"name"` // Format: tasks/{task_id}/pushNotificationConfigs/{config_id}
}

// CreateTaskPushNotificationConfigRequest represents a request to create push notification config
type CreateTaskPushNotificationConfigRequest struct {
	Parent   string                      `json:"parent"` // Format: tasks/{task_id}
	ConfigID string                      `json:"config_id"`
	Config   *TaskPushNotificationConfig `json:"config"`
}

// TaskSubscriptionRequest represents a request to subscribe to task updates
type TaskSubscriptionRequest struct {
	Name string `json:"name"` // Format: tasks/{task_id}
}

// ListTaskPushNotificationConfigRequest represents a request to list push notification configs
type ListTaskPushNotificationConfigRequest struct {
	Parent    string `json:"parent"` // Format: tasks/{task_id}
	PageSize  int32  `json:"page_size,omitempty"`
	PageToken string `json:"page_token,omitempty"`
}

// GetAgentCardRequest represents a request to get agent card information
type GetAgentCardRequest struct {
	// Empty request - no fields needed
}

// ListTasksRequest represents a request to list tasks
type ListTasksRequest struct {
	ContextID string      `json:"contextId,omitempty"`
	States    []TaskState `json:"states,omitempty"`
	PageSize  int32       `json:"pageSize,omitempty"`
	PageToken string      `json:"pageToken,omitempty"`
}

// SendMessageRequestToProto converts SendMessageRequest to pb.SendMessageRequest
func SendMessageRequestToProto(req *SendMessageRequest) (*pb.SendMessageRequest, error) {
	if req == nil {
		return nil, nil
	}

	message, err := MessageToProto(req.Request)
	if err != nil {
		return nil, fmt.Errorf("converting request message: %w", err)
	}

	var config *pb.SendMessageConfiguration
	if req.Configuration != nil {
		config, err = SendMessageConfigurationToProto(req.Configuration)
		if err != nil {
			return nil, fmt.Errorf("converting configuration: %w", err)
		}
	}

	metadata, err := MapToStruct(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting metadata: %w", err)
	}

	return &pb.SendMessageRequest{
		Request:       message,
		Configuration: config,
		Metadata:      metadata,
	}, nil
}

// SendMessageRequestFromProto converts pb.SendMessageRequest to SendMessageRequest
func SendMessageRequestFromProto(req *pb.SendMessageRequest) (*SendMessageRequest, error) {
	if req == nil {
		return nil, nil
	}

	message, err := MessageFromProto(req.Request)
	if err != nil {
		return nil, fmt.Errorf("converting request message: %w", err)
	}

	var config *SendMessageConfiguration
	if req.Configuration != nil {
		config, err = SendMessageConfigurationFromProto(req.Configuration)
		if err != nil {
			return nil, fmt.Errorf("converting configuration: %w", err)
		}
	}

	metadata, err := StructToMap(req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting metadata: %w", err)
	}

	return &SendMessageRequest{
		Request:       message,
		Configuration: config,
		Metadata:      metadata,
	}, nil
}

// GetTaskRequestToProto converts GetTaskRequest to pb.GetTaskRequest
func GetTaskRequestToProto(req *GetTaskRequest) *pb.GetTaskRequest {
	if req == nil {
		return nil
	}

	return &pb.GetTaskRequest{
		Name:          req.Name,
		HistoryLength: req.HistoryLength,
	}
}

// GetTaskRequestFromProto converts pb.GetTaskRequest to GetTaskRequest
func GetTaskRequestFromProto(req *pb.GetTaskRequest) *GetTaskRequest {
	if req == nil {
		return nil
	}

	return &GetTaskRequest{
		Name:          req.Name,
		HistoryLength: req.HistoryLength,
	}
}

// CancelTaskRequestToProto converts CancelTaskRequest to pb.CancelTaskRequest
func CancelTaskRequestToProto(req *CancelTaskRequest) *pb.CancelTaskRequest {
	if req == nil {
		return nil
	}

	return &pb.CancelTaskRequest{
		Name: req.Name,
	}
}

// CancelTaskRequestFromProto converts pb.CancelTaskRequest to CancelTaskRequest
func CancelTaskRequestFromProto(req *pb.CancelTaskRequest) *CancelTaskRequest {
	if req == nil {
		return nil
	}

	return &CancelTaskRequest{
		Name: req.Name,
	}
}

// GetTaskPushNotificationConfigRequestToProto converts GetTaskPushNotificationConfigRequest to pb.GetTaskPushNotificationConfigRequest
func GetTaskPushNotificationConfigRequestToProto(req *GetTaskPushNotificationConfigRequest) *pb.GetTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &pb.GetTaskPushNotificationConfigRequest{
		Name: req.Name,
	}
}

// GetTaskPushNotificationConfigRequestFromProto converts pb.GetTaskPushNotificationConfigRequest to GetTaskPushNotificationConfigRequest
func GetTaskPushNotificationConfigRequestFromProto(req *pb.GetTaskPushNotificationConfigRequest) *GetTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &GetTaskPushNotificationConfigRequest{
		Name: req.Name,
	}
}

// DeleteTaskPushNotificationConfigRequestToProto converts DeleteTaskPushNotificationConfigRequest to pb.DeleteTaskPushNotificationConfigRequest
func DeleteTaskPushNotificationConfigRequestToProto(req *DeleteTaskPushNotificationConfigRequest) *pb.DeleteTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &pb.DeleteTaskPushNotificationConfigRequest{
		Name: req.Name,
	}
}

// DeleteTaskPushNotificationConfigRequestFromProto converts pb.DeleteTaskPushNotificationConfigRequest to DeleteTaskPushNotificationConfigRequest
func DeleteTaskPushNotificationConfigRequestFromProto(req *pb.DeleteTaskPushNotificationConfigRequest) *DeleteTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &DeleteTaskPushNotificationConfigRequest{
		Name: req.Name,
	}
}

// CreateTaskPushNotificationConfigRequestToProto converts CreateTaskPushNotificationConfigRequest to pb.CreateTaskPushNotificationConfigRequest
func CreateTaskPushNotificationConfigRequestToProto(req *CreateTaskPushNotificationConfigRequest) (*pb.CreateTaskPushNotificationConfigRequest, error) {
	if req == nil {
		return nil, nil
	}

	config, err := TaskPushNotificationConfigToProto(req.Config)
	if err != nil {
		return nil, fmt.Errorf("converting config: %w", err)
	}

	return &pb.CreateTaskPushNotificationConfigRequest{
		Parent:   req.Parent,
		ConfigId: req.ConfigID,
		Config:   config,
	}, nil
}

// CreateTaskPushNotificationConfigRequestFromProto converts pb.CreateTaskPushNotificationConfigRequest to CreateTaskPushNotificationConfigRequest
func CreateTaskPushNotificationConfigRequestFromProto(req *pb.CreateTaskPushNotificationConfigRequest) (*CreateTaskPushNotificationConfigRequest, error) {
	if req == nil {
		return nil, nil
	}

	config, err := TaskPushNotificationConfigFromProto(req.Config)
	if err != nil {
		return nil, fmt.Errorf("converting config: %w", err)
	}

	return &CreateTaskPushNotificationConfigRequest{
		Parent:   req.Parent,
		ConfigID: req.ConfigId,
		Config:   config,
	}, nil
}

// TaskSubscriptionRequestToProto converts TaskSubscriptionRequest to pb.TaskSubscriptionRequest
func TaskSubscriptionRequestToProto(req *TaskSubscriptionRequest) *pb.TaskSubscriptionRequest {
	if req == nil {
		return nil
	}

	return &pb.TaskSubscriptionRequest{
		Name: req.Name,
	}
}

// TaskSubscriptionRequestFromProto converts pb.TaskSubscriptionRequest to TaskSubscriptionRequest
func TaskSubscriptionRequestFromProto(req *pb.TaskSubscriptionRequest) *TaskSubscriptionRequest {
	if req == nil {
		return nil
	}

	return &TaskSubscriptionRequest{
		Name: req.Name,
	}
}

// ListTaskPushNotificationConfigRequestToProto converts ListTaskPushNotificationConfigRequest to pb.ListTaskPushNotificationConfigRequest
func ListTaskPushNotificationConfigRequestToProto(req *ListTaskPushNotificationConfigRequest) *pb.ListTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &pb.ListTaskPushNotificationConfigRequest{
		Parent:    req.Parent,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}
}

// ListTaskPushNotificationConfigRequestFromProto converts pb.ListTaskPushNotificationConfigRequest to ListTaskPushNotificationConfigRequest
func ListTaskPushNotificationConfigRequestFromProto(req *pb.ListTaskPushNotificationConfigRequest) *ListTaskPushNotificationConfigRequest {
	if req == nil {
		return nil
	}

	return &ListTaskPushNotificationConfigRequest{
		Parent:    req.Parent,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}
}

// GetAgentCardRequestToProto converts GetAgentCardRequest to pb.GetAgentCardRequest
func GetAgentCardRequestToProto(req *GetAgentCardRequest) *pb.GetAgentCardRequest {
	if req == nil {
		return nil
	}

	return &pb.GetAgentCardRequest{}
}

// GetAgentCardRequestFromProto converts pb.GetAgentCardRequest to GetAgentCardRequest
func GetAgentCardRequestFromProto(req *pb.GetAgentCardRequest) *GetAgentCardRequest {
	if req == nil {
		return nil
	}

	return &GetAgentCardRequest{}
}

// ListTasksRequestToProto converts ListTasksRequest to pb.ListTasksRequest
func ListTasksRequestToProto(req *ListTasksRequest) *pb.ListTasksRequest {
	if req == nil {
		return nil
	}

	states := make([]pb.TaskState, len(req.States))
	for i, state := range req.States {
		states[i] = TaskStateToProto(state)
	}

	return &pb.ListTasksRequest{
		ContextId: req.ContextID,
		States:    states,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}
}

// ListTasksRequestFromProto converts pb.ListTasksRequest to ListTasksRequest
func ListTasksRequestFromProto(req *pb.ListTasksRequest) *ListTasksRequest {
	if req == nil {
		return nil
	}

	states := make([]TaskState, len(req.States))
	for i, state := range req.States {
		states[i] = TaskStateFromProto(state)
	}

	return &ListTasksRequest{
		ContextID: req.ContextId,
		States:    states,
		PageSize:  req.PageSize,
		PageToken: req.PageToken,
	}
}
