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

// SendMessageConfiguration contains configuration for a send message request
type SendMessageConfiguration struct {
	AcceptedOutputModes []string                `json:"acceptedOutputModes,omitempty"`
	PushNotification    *PushNotificationConfig `json:"pushNotification,omitempty"`
	HistoryLength       int32                   `json:"historyLength,omitempty"`
	Blocking            bool                    `json:"blocking,omitempty"`
}

// PushNotificationConfig contains configuration for push notifications
type PushNotificationConfig struct {
	ID             string              `json:"id"`
	URL            string              `json:"url"`
	Token          string              `json:"token"`
	Authentication *AuthenticationInfo `json:"authentication,omitempty"`
}

// AuthenticationInfo contains authentication details for push notifications
type AuthenticationInfo struct {
	Schemes     []string `json:"schemes,omitempty"`
	Credentials string   `json:"credentials,omitempty"`
}

// TaskPushNotificationConfig represents a push notification configuration for a specific task
type TaskPushNotificationConfig struct {
	Name                   string                  `json:"name"`
	PushNotificationConfig *PushNotificationConfig `json:"pushNotificationConfig"`
}

// SendMessageConfigurationToProto converts SendMessageConfiguration to pb.SendMessageConfiguration
func SendMessageConfigurationToProto(config *SendMessageConfiguration) (*pb.SendMessageConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	var pushNotification *pb.PushNotificationConfig
	var err error
	if config.PushNotification != nil {
		pushNotification, err = PushNotificationConfigToProto(config.PushNotification)
		if err != nil {
			return nil, fmt.Errorf("converting push notification config: %w", err)
		}
	}

	return &pb.SendMessageConfiguration{
		AcceptedOutputModes: config.AcceptedOutputModes,
		PushNotification:    pushNotification,
		HistoryLength:       config.HistoryLength,
		Blocking:            config.Blocking,
	}, nil
}

// SendMessageConfigurationFromProto converts pb.SendMessageConfiguration to SendMessageConfiguration
func SendMessageConfigurationFromProto(config *pb.SendMessageConfiguration) (*SendMessageConfiguration, error) {
	if config == nil {
		return nil, nil
	}

	var pushNotification *PushNotificationConfig
	var err error
	if config.PushNotification != nil {
		pushNotification, err = PushNotificationConfigFromProto(config.PushNotification)
		if err != nil {
			return nil, fmt.Errorf("converting push notification config: %w", err)
		}
	}

	return &SendMessageConfiguration{
		AcceptedOutputModes: config.AcceptedOutputModes,
		PushNotification:    pushNotification,
		HistoryLength:       config.HistoryLength,
		Blocking:            config.Blocking,
	}, nil
}

// PushNotificationConfigToProto converts PushNotificationConfig to pb.PushNotificationConfig
func PushNotificationConfigToProto(config *PushNotificationConfig) (*pb.PushNotificationConfig, error) {
	if config == nil {
		return nil, nil
	}

	var auth *pb.AuthenticationInfo
	var err error
	if config.Authentication != nil {
		auth, err = AuthenticationInfoToProto(config.Authentication)
		if err != nil {
			return nil, fmt.Errorf("converting authentication info: %w", err)
		}
	}

	return &pb.PushNotificationConfig{
		Id:             config.ID,
		Url:            config.URL,
		Token:          config.Token,
		Authentication: auth,
	}, nil
}

// PushNotificationConfigFromProto converts pb.PushNotificationConfig to PushNotificationConfig
func PushNotificationConfigFromProto(config *pb.PushNotificationConfig) (*PushNotificationConfig, error) {
	if config == nil {
		return nil, nil
	}

	var auth *AuthenticationInfo
	var err error
	if config.Authentication != nil {
		auth, err = AuthenticationInfoFromProto(config.Authentication)
		if err != nil {
			return nil, fmt.Errorf("converting authentication info: %w", err)
		}
	}

	return &PushNotificationConfig{
		ID:             config.Id,
		URL:            config.Url,
		Token:          config.Token,
		Authentication: auth,
	}, nil
}

// AuthenticationInfoToProto converts AuthenticationInfo to pb.AuthenticationInfo
func AuthenticationInfoToProto(auth *AuthenticationInfo) (*pb.AuthenticationInfo, error) {
	if auth == nil {
		return nil, nil
	}

	return &pb.AuthenticationInfo{
		Schemes:     auth.Schemes,
		Credentials: auth.Credentials,
	}, nil
}

// AuthenticationInfoFromProto converts pb.AuthenticationInfo to AuthenticationInfo
func AuthenticationInfoFromProto(auth *pb.AuthenticationInfo) (*AuthenticationInfo, error) {
	if auth == nil {
		return nil, nil
	}

	return &AuthenticationInfo{
		Schemes:     auth.Schemes,
		Credentials: auth.Credentials,
	}, nil
}

// TaskPushNotificationConfigToProto converts TaskPushNotificationConfig to pb.TaskPushNotificationConfig
func TaskPushNotificationConfigToProto(config *TaskPushNotificationConfig) (*pb.TaskPushNotificationConfig, error) {
	if config == nil {
		return nil, nil
	}

	pushConfig, err := PushNotificationConfigToProto(config.PushNotificationConfig)
	if err != nil {
		return nil, fmt.Errorf("converting push notification config: %w", err)
	}

	return &pb.TaskPushNotificationConfig{
		Name:                   config.Name,
		PushNotificationConfig: pushConfig,
	}, nil
}

// TaskPushNotificationConfigFromProto converts pb.TaskPushNotificationConfig to TaskPushNotificationConfig
func TaskPushNotificationConfigFromProto(config *pb.TaskPushNotificationConfig) (*TaskPushNotificationConfig, error) {
	if config == nil {
		return nil, nil
	}

	pushConfig, err := PushNotificationConfigFromProto(config.PushNotificationConfig)
	if err != nil {
		return nil, fmt.Errorf("converting push notification config: %w", err)
	}

	return &TaskPushNotificationConfig{
		Name:                   config.Name,
		PushNotificationConfig: pushConfig,
	}, nil
}
