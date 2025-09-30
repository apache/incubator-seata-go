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

// TaskStatusUpdateEvent represents a delta event on a task indicating that a task has changed
type TaskStatusUpdateEvent struct {
	TaskID    string         `json:"taskId"`
	ContextID string         `json:"contextId"`
	Status    *TaskStatus    `json:"status"`
	Final     bool           `json:"final"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TaskArtifactUpdateEvent represents a task delta where an artifact has been generated
type TaskArtifactUpdateEvent struct {
	TaskID    string         `json:"taskId"`
	ContextID string         `json:"contextId"`
	Artifact  *Artifact      `json:"artifact"`
	Append    bool           `json:"append"`
	LastChunk bool           `json:"lastChunk"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// StreamResponse represents the different types of responses in a stream
type StreamResponse interface {
	StreamResponseType() StreamResponseTypeEnum
}

// StreamResponseTypeEnum identifies the type of stream response
type StreamResponseTypeEnum int

const (
	StreamResponseTypeTask StreamResponseTypeEnum = iota
	StreamResponseTypeMessage
	StreamResponseTypeStatusUpdate
	StreamResponseTypeArtifactUpdate
)

// TaskStreamResponse represents a task in a stream response
type TaskStreamResponse struct {
	Task *Task `json:"task"`
}

func (t *TaskStreamResponse) StreamResponseType() StreamResponseTypeEnum {
	return StreamResponseTypeTask
}

// MessageStreamResponse represents a message in a stream response
type MessageStreamResponse struct {
	Message *Message `json:"message"`
}

func (m *MessageStreamResponse) StreamResponseType() StreamResponseTypeEnum {
	return StreamResponseTypeMessage
}

// StatusUpdateStreamResponse represents a status update in a stream response
type StatusUpdateStreamResponse struct {
	StatusUpdate *TaskStatusUpdateEvent `json:"status_update"`
}

func (s *StatusUpdateStreamResponse) StreamResponseType() StreamResponseTypeEnum {
	return StreamResponseTypeStatusUpdate
}

// ArtifactUpdateStreamResponse represents an artifact update in a stream response
type ArtifactUpdateStreamResponse struct {
	ArtifactUpdate *TaskArtifactUpdateEvent `json:"artifact_update"`
}

func (a *ArtifactUpdateStreamResponse) StreamResponseType() StreamResponseTypeEnum {
	return StreamResponseTypeArtifactUpdate
}

// TaskStatusUpdateEventToProto converts TaskStatusUpdateEvent to pb.TaskStatusUpdateEvent
func TaskStatusUpdateEventToProto(event *TaskStatusUpdateEvent) (*pb.TaskStatusUpdateEvent, error) {
	if event == nil {
		return nil, nil
	}

	status, err := TaskStatusToProto(event.Status)
	if err != nil {
		return nil, fmt.Errorf("converting task status: %w", err)
	}

	metadata, err := MapToStruct(event.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting event metadata: %w", err)
	}

	return &pb.TaskStatusUpdateEvent{
		TaskId:    event.TaskID,
		ContextId: event.ContextID,
		Status:    status,
		Final:     event.Final,
		Metadata:  metadata,
	}, nil
}

// TaskStatusUpdateEventFromProto converts pb.TaskStatusUpdateEvent to TaskStatusUpdateEvent
func TaskStatusUpdateEventFromProto(event *pb.TaskStatusUpdateEvent) (*TaskStatusUpdateEvent, error) {
	if event == nil {
		return nil, nil
	}

	status, err := TaskStatusFromProto(event.Status)
	if err != nil {
		return nil, fmt.Errorf("converting task status: %w", err)
	}

	metadata, err := StructToMap(event.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting event metadata: %w", err)
	}

	return &TaskStatusUpdateEvent{
		TaskID:    event.TaskId,
		ContextID: event.ContextId,
		Status:    status,
		Final:     event.Final,
		Metadata:  metadata,
	}, nil
}

// TaskArtifactUpdateEventToProto converts TaskArtifactUpdateEvent to pb.TaskArtifactUpdateEvent
func TaskArtifactUpdateEventToProto(event *TaskArtifactUpdateEvent) (*pb.TaskArtifactUpdateEvent, error) {
	if event == nil {
		return nil, nil
	}

	artifact, err := ArtifactToProto(event.Artifact)
	if err != nil {
		return nil, fmt.Errorf("converting artifact: %w", err)
	}

	metadata, err := MapToStruct(event.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting event metadata: %w", err)
	}

	return &pb.TaskArtifactUpdateEvent{
		TaskId:    event.TaskID,
		ContextId: event.ContextID,
		Artifact:  artifact,
		Append:    event.Append,
		LastChunk: event.LastChunk,
		Metadata:  metadata,
	}, nil
}

// TaskArtifactUpdateEventFromProto converts pb.TaskArtifactUpdateEvent to TaskArtifactUpdateEvent
func TaskArtifactUpdateEventFromProto(event *pb.TaskArtifactUpdateEvent) (*TaskArtifactUpdateEvent, error) {
	if event == nil {
		return nil, nil
	}

	artifact, err := ArtifactFromProto(event.Artifact)
	if err != nil {
		return nil, fmt.Errorf("converting artifact: %w", err)
	}

	metadata, err := StructToMap(event.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting event metadata: %w", err)
	}

	return &TaskArtifactUpdateEvent{
		TaskID:    event.TaskId,
		ContextID: event.ContextId,
		Artifact:  artifact,
		Append:    event.Append,
		LastChunk: event.LastChunk,
		Metadata:  metadata,
	}, nil
}

// StreamResponseToProto converts StreamResponse to pb.StreamResponse
func StreamResponseToProto(response StreamResponse) (*pb.StreamResponse, error) {
	if response == nil {
		return nil, nil
	}

	pbResponse := &pb.StreamResponse{}

	switch r := response.(type) {
	case *TaskStreamResponse:
		task, err := TaskToProto(r.Task)
		if err != nil {
			return nil, fmt.Errorf("converting task stream response: %w", err)
		}
		pbResponse.Payload = &pb.StreamResponse_Task{Task: task}
	case *MessageStreamResponse:
		msg, err := MessageToProto(r.Message)
		if err != nil {
			return nil, fmt.Errorf("converting message stream response: %w", err)
		}
		pbResponse.Payload = &pb.StreamResponse_Msg{Msg: msg}
	case *StatusUpdateStreamResponse:
		statusUpdate, err := TaskStatusUpdateEventToProto(r.StatusUpdate)
		if err != nil {
			return nil, fmt.Errorf("converting status update stream response: %w", err)
		}
		pbResponse.Payload = &pb.StreamResponse_StatusUpdate{StatusUpdate: statusUpdate}
	case *ArtifactUpdateStreamResponse:
		artifactUpdate, err := TaskArtifactUpdateEventToProto(r.ArtifactUpdate)
		if err != nil {
			return nil, fmt.Errorf("converting artifact update stream response: %w", err)
		}
		pbResponse.Payload = &pb.StreamResponse_ArtifactUpdate{ArtifactUpdate: artifactUpdate}
	default:
		return nil, fmt.Errorf("unknown stream response type: %T", response)
	}

	return pbResponse, nil
}

// StreamResponseFromProto converts pb.StreamResponse to StreamResponse
func StreamResponseFromProto(response *pb.StreamResponse) (StreamResponse, error) {
	if response == nil {
		return nil, nil
	}

	switch payload := response.Payload.(type) {
	case *pb.StreamResponse_Task:
		task, err := TaskFromProto(payload.Task)
		if err != nil {
			return nil, fmt.Errorf("converting task stream response: %w", err)
		}
		return &TaskStreamResponse{Task: task}, nil
	case *pb.StreamResponse_Msg:
		msg, err := MessageFromProto(payload.Msg)
		if err != nil {
			return nil, fmt.Errorf("converting message stream response: %w", err)
		}
		return &MessageStreamResponse{Message: msg}, nil
	case *pb.StreamResponse_StatusUpdate:
		statusUpdate, err := TaskStatusUpdateEventFromProto(payload.StatusUpdate)
		if err != nil {
			return nil, fmt.Errorf("converting status update stream response: %w", err)
		}
		return &StatusUpdateStreamResponse{StatusUpdate: statusUpdate}, nil
	case *pb.StreamResponse_ArtifactUpdate:
		artifactUpdate, err := TaskArtifactUpdateEventFromProto(payload.ArtifactUpdate)
		if err != nil {
			return nil, fmt.Errorf("converting artifact update stream response: %w", err)
		}
		return &ArtifactUpdateStreamResponse{ArtifactUpdate: artifactUpdate}, nil
	default:
		return nil, fmt.Errorf("unknown stream response payload type: %T", payload)
	}
}
