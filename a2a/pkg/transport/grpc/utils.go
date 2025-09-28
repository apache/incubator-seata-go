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
	"errors"
	"fmt"
	"strings"

	"seata-go-ai-a2a/pkg/types"

	pb "seata-go-ai-a2a/pkg/proto/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// extractTaskIDFromName extracts task ID from resource name format (tasks/{task_id})
func extractTaskIDFromName(name string) string {
	if !strings.HasPrefix(name, "tasks/") {
		return ""
	}
	return strings.TrimPrefix(name, "tasks/")
}

// extractConfigIDFromName extracts config ID from resource name format (tasks/{task_id}/pushNotificationConfigs/{config_id})
func extractConfigIDFromName(name string) (taskID, configID string) {
	parts := strings.Split(name, "/")
	if len(parts) >= 4 && parts[0] == "tasks" && parts[2] == "pushNotificationConfigs" {
		return parts[1], parts[3]
	}
	return "", ""
}

// isA2AError checks if an error is an A2A-specific error
func isA2AError(err error) bool {
	var a2AError *types.A2AError
	switch {
	case errors.As(err, &a2AError):
		return true
	default:
		return false
	}
}

// mapA2AErrorToGRPCStatus maps A2A error codes to gRPC status codes
func mapA2AErrorToGRPCStatus(err error) error {
	a2aErr, ok := err.(*types.A2AError)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}

	switch a2aErr.Code {
	case types.TaskNotFoundError:
		return status.Error(codes.NotFound, a2aErr.Message)
	case types.TaskNotCancelableError:
		return status.Error(codes.FailedPrecondition, a2aErr.Message)
	case types.UnsupportedOperationError:
		return status.Error(codes.Unimplemented, a2aErr.Message)
	case types.ContentTypeNotSupportedError:
		return status.Error(codes.InvalidArgument, a2aErr.Message)
	case types.InvalidAgentResponseError:
		return status.Error(codes.InvalidArgument, a2aErr.Message)
	case types.AuthenticatedExtendedCardNotConfiguredError:
		return status.Error(codes.FailedPrecondition, a2aErr.Message)
	default:
		return status.Error(codes.Internal, a2aErr.Message)
	}
}

// convertStreamResponseToProto converts a types.StreamResponse to pb.StreamResponse
func convertStreamResponseToProto(resp types.StreamResponse) (*pb.StreamResponse, error) {
	switch r := resp.(type) {
	case *types.TaskStreamResponse:
		pbTask, err := types.TaskToProto(r.Task)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task: %w", err)
		}
		return &pb.StreamResponse{
			Payload: &pb.StreamResponse_Task{Task: pbTask},
		}, nil

	case *types.MessageStreamResponse:
		pbMessage, err := types.MessageToProto(r.Message)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		return &pb.StreamResponse{
			Payload: &pb.StreamResponse_Msg{Msg: pbMessage},
		}, nil

	case *types.StatusUpdateStreamResponse:
		pbStatusUpdate, err := types.TaskStatusUpdateEventToProto(r.StatusUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert status update: %w", err)
		}
		return &pb.StreamResponse{
			Payload: &pb.StreamResponse_StatusUpdate{StatusUpdate: pbStatusUpdate},
		}, nil

	case *types.ArtifactUpdateStreamResponse:
		pbArtifactUpdate, err := types.TaskArtifactUpdateEventToProto(r.ArtifactUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert artifact update: %w", err)
		}
		return &pb.StreamResponse{
			Payload: &pb.StreamResponse_ArtifactUpdate{ArtifactUpdate: pbArtifactUpdate},
		}, nil

	default:
		return nil, fmt.Errorf("unknown stream response type: %T", resp)
	}
}

// convertStreamResponseFromProto converts a pb.StreamResponse to types.StreamResponse
func convertStreamResponseFromProto(pbResp *pb.StreamResponse) (types.StreamResponse, error) {
	switch resp := pbResp.Payload.(type) {
	case *pb.StreamResponse_Task:
		task, err := types.TaskFromProto(resp.Task)
		if err != nil {
			return nil, fmt.Errorf("failed to convert task: %w", err)
		}
		return &types.TaskStreamResponse{Task: task}, nil

	case *pb.StreamResponse_Msg:
		message, err := types.MessageFromProto(resp.Msg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert message: %w", err)
		}
		return &types.MessageStreamResponse{Message: message}, nil

	case *pb.StreamResponse_StatusUpdate:
		statusUpdate, err := types.TaskStatusUpdateEventFromProto(resp.StatusUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert status update: %w", err)
		}
		return &types.StatusUpdateStreamResponse{StatusUpdate: statusUpdate}, nil

	case *pb.StreamResponse_ArtifactUpdate:
		artifactUpdate, err := types.TaskArtifactUpdateEventFromProto(resp.ArtifactUpdate)
		if err != nil {
			return nil, fmt.Errorf("failed to convert artifact update: %w", err)
		}
		return &types.ArtifactUpdateStreamResponse{ArtifactUpdate: artifactUpdate}, nil

	default:
		return nil, fmt.Errorf("unknown protobuf stream response type: %T", resp)
	}
}

// validateGRPCRequest performs basic validation on gRPC requests
func validateGRPCRequest(req interface{}) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request cannot be nil")
	}
	return nil
}

// validateTaskName validates that a task name follows the expected format
func validateTaskName(name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "task name cannot be empty")
	}
	if !strings.HasPrefix(name, "tasks/") {
		return status.Error(codes.InvalidArgument, "task name must follow format: tasks/{task_id}")
	}
	taskID := strings.TrimPrefix(name, "tasks/")
	if taskID == "" {
		return status.Error(codes.InvalidArgument, "task ID cannot be empty")
	}
	return nil
}

// validatePushNotificationConfigName validates push notification config name format
func validatePushNotificationConfigName(name string) error {
	if name == "" {
		return status.Error(codes.InvalidArgument, "config name cannot be empty")
	}

	parts := strings.Split(name, "/")
	if len(parts) != 4 || parts[0] != "tasks" || parts[2] != "pushNotificationConfigs" {
		return status.Error(codes.InvalidArgument, "config name must follow format: tasks/{task_id}/pushNotificationConfigs/{config_id}")
	}

	if parts[1] == "" {
		return status.Error(codes.InvalidArgument, "task ID cannot be empty")
	}
	if parts[3] == "" {
		return status.Error(codes.InvalidArgument, "config ID cannot be empty")
	}

	return nil
}

// createGRPCStatusFromError creates a gRPC status from a standard error
func createGRPCStatusFromError(err error, defaultCode codes.Code) error {
	if err == nil {
		return nil
	}

	if isA2AError(err) {
		return mapA2AErrorToGRPCStatus(err)
	}

	return status.Error(defaultCode, err.Error())
}
