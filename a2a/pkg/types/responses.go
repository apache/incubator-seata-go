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

	"google.golang.org/protobuf/types/known/emptypb"
)

// SendMessageResponse represents a response to a send message request
type SendMessageResponse interface {
	SendMessageResponseType() SendMessageResponseTypeEnum
}

// SendMessageResponseTypeEnum identifies the type of send message response
type SendMessageResponseTypeEnum int

const (
	SendMessageResponseTypeTask SendMessageResponseTypeEnum = iota
	SendMessageResponseTypeMessage
)

// TaskSendMessageResponse represents a task response to a send message request
type TaskSendMessageResponse struct {
	Task *Task `json:"task"`
}

func (t *TaskSendMessageResponse) SendMessageResponseType() SendMessageResponseTypeEnum {
	return SendMessageResponseTypeTask
}

// MessageSendMessageResponse represents a message response to a send message request
type MessageSendMessageResponse struct {
	Message *Message `json:"message"`
}

func (m *MessageSendMessageResponse) SendMessageResponseType() SendMessageResponseTypeEnum {
	return SendMessageResponseTypeMessage
}

// ListTaskPushNotificationConfigResponse represents a response to list push notification configs
type ListTaskPushNotificationConfigResponse struct {
	Configs       []*TaskPushNotificationConfig `json:"configs"`
	NextPageToken string                        `json:"next_page_token,omitempty"`
}

// DeleteTaskPushNotificationConfigResponse represents a response to delete push notification config
type DeleteTaskPushNotificationConfigResponse struct {
	// Empty response
}

// ListTasksResponse represents a response to list tasks request
type ListTasksResponse struct {
	Tasks         []*Task `json:"tasks"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
}

// SendMessageResponseToProto converts SendMessageResponse to pb.SendMessageResponse
func SendMessageResponseToProto(response SendMessageResponse) (*pb.SendMessageResponse, error) {
	if response == nil {
		return nil, nil
	}

	pbResponse := &pb.SendMessageResponse{}

	switch r := response.(type) {
	case *TaskSendMessageResponse:
		task, err := TaskToProto(r.Task)
		if err != nil {
			return nil, fmt.Errorf("converting task response: %w", err)
		}
		pbResponse.Payload = &pb.SendMessageResponse_Task{Task: task}
	case *MessageSendMessageResponse:
		msg, err := MessageToProto(r.Message)
		if err != nil {
			return nil, fmt.Errorf("converting message response: %w", err)
		}
		pbResponse.Payload = &pb.SendMessageResponse_Msg{Msg: msg}
	default:
		return nil, fmt.Errorf("unknown send message response type: %T", response)
	}

	return pbResponse, nil
}

// SendMessageResponseFromProto converts pb.SendMessageResponse to SendMessageResponse
func SendMessageResponseFromProto(response *pb.SendMessageResponse) (SendMessageResponse, error) {
	if response == nil {
		return nil, nil
	}

	switch payload := response.Payload.(type) {
	case *pb.SendMessageResponse_Task:
		task, err := TaskFromProto(payload.Task)
		if err != nil {
			return nil, fmt.Errorf("converting task response: %w", err)
		}
		return &TaskSendMessageResponse{Task: task}, nil
	case *pb.SendMessageResponse_Msg:
		msg, err := MessageFromProto(payload.Msg)
		if err != nil {
			return nil, fmt.Errorf("converting message response: %w", err)
		}
		return &MessageSendMessageResponse{Message: msg}, nil
	default:
		return nil, fmt.Errorf("unknown send message response payload type: %T", payload)
	}
}

// ListTaskPushNotificationConfigResponseToProto converts ListTaskPushNotificationConfigResponse to pb.ListTaskPushNotificationConfigResponse
func ListTaskPushNotificationConfigResponseToProto(response *ListTaskPushNotificationConfigResponse) (*pb.ListTaskPushNotificationConfigResponse, error) {
	if response == nil {
		return nil, nil
	}

	configs := make([]*pb.TaskPushNotificationConfig, len(response.Configs))
	for i, config := range response.Configs {
		var err error
		configs[i], err = TaskPushNotificationConfigToProto(config)
		if err != nil {
			return nil, fmt.Errorf("converting config %d: %w", i, err)
		}
	}

	return &pb.ListTaskPushNotificationConfigResponse{
		Configs:       configs,
		NextPageToken: response.NextPageToken,
	}, nil
}

// ListTaskPushNotificationConfigResponseFromProto converts pb.ListTaskPushNotificationConfigResponse to ListTaskPushNotificationConfigResponse
func ListTaskPushNotificationConfigResponseFromProto(response *pb.ListTaskPushNotificationConfigResponse) (*ListTaskPushNotificationConfigResponse, error) {
	if response == nil {
		return nil, nil
	}

	configs := make([]*TaskPushNotificationConfig, len(response.Configs))
	for i, config := range response.Configs {
		var err error
		configs[i], err = TaskPushNotificationConfigFromProto(config)
		if err != nil {
			return nil, fmt.Errorf("converting config %d: %w", i, err)
		}
	}

	return &ListTaskPushNotificationConfigResponse{
		Configs:       configs,
		NextPageToken: response.NextPageToken,
	}, nil
}

// DeleteTaskPushNotificationConfigResponseToProto converts DeleteTaskPushNotificationConfigResponse to emptypb.Empty
func DeleteTaskPushNotificationConfigResponseToProto(response *DeleteTaskPushNotificationConfigResponse) *emptypb.Empty {
	return &emptypb.Empty{}
}

// DeleteTaskPushNotificationConfigResponseFromProto converts emptypb.Empty to DeleteTaskPushNotificationConfigResponse
func DeleteTaskPushNotificationConfigResponseFromProto(response *emptypb.Empty) *DeleteTaskPushNotificationConfigResponse {
	return &DeleteTaskPushNotificationConfigResponse{}
}

// ListTasksResponseToProto converts ListTasksResponse to pb.ListTasksResponse
func ListTasksResponseToProto(response *ListTasksResponse) (*pb.ListTasksResponse, error) {
	if response == nil {
		return nil, nil
	}

	tasks := make([]*pb.Task, len(response.Tasks))
	for i, task := range response.Tasks {
		var err error
		tasks[i], err = TaskToProto(task)
		if err != nil {
			return nil, fmt.Errorf("converting task %d: %w", i, err)
		}
	}

	return &pb.ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: response.NextPageToken,
	}, nil
}

// ListTasksResponseFromProto converts pb.ListTasksResponse to ListTasksResponse
func ListTasksResponseFromProto(response *pb.ListTasksResponse) (*ListTasksResponse, error) {
	if response == nil {
		return nil, nil
	}

	tasks := make([]*Task, len(response.Tasks))
	for i, task := range response.Tasks {
		var err error
		tasks[i], err = TaskFromProto(task)
		if err != nil {
			return nil, fmt.Errorf("converting task %d: %w", i, err)
		}
	}

	return &ListTasksResponse{
		Tasks:         tasks,
		NextPageToken: response.NextPageToken,
	}, nil
}
