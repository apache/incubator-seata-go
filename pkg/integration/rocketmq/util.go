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

package rocketmq

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"seata.apache.org/seata-go/pkg/util/log"
)

type MessageBuilder struct {
	message *primitive.Message
}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		message: &primitive.Message{},
	}
}

func (b *MessageBuilder) WithTopic(topic string) *MessageBuilder {
	b.message.Topic = topic
	return b
}

func (b *MessageBuilder) WithTags(tags string) *MessageBuilder {
	b.message.WithTag(tags)
	return b
}

func (b *MessageBuilder) WithKeys(keys ...string) *MessageBuilder {
	b.message.WithKeys(keys)
	return b
}

func (b *MessageBuilder) WithBody(body []byte) *MessageBuilder {
	b.message.Body = body
	return b
}

func (b *MessageBuilder) WithJSONBody(obj interface{}) *MessageBuilder {
	data, err := json.Marshal(obj)
	if err != nil {
		log.Errorf("Failed to marshal JSON body: %v", err)
		return b
	}
	b.message.Body = data
	return b
}

func (b *MessageBuilder) WithProperty(key, value string) *MessageBuilder {
	b.message.WithProperty(key, value)
	return b
}

func (b *MessageBuilder) WithProperties(properties map[string]string) *MessageBuilder {
	b.message.WithProperties(properties)
	return b
}

func (b *MessageBuilder) WithDelayLevel(level int) *MessageBuilder {
	b.message.WithDelayTimeLevel(level)
	return b
}

func (b *MessageBuilder) Build() *primitive.Message {
	return b.message
}

type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	Multiplier float64
	Jitter     bool
}

func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

type MessageValidator struct{}

func NewMessageValidator() *MessageValidator {
	return &MessageValidator{}
}

func (v *MessageValidator) Validate(msg *primitive.Message) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}

	if strings.TrimSpace(msg.Topic) == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	if len(msg.Body) == 0 {
		return fmt.Errorf("message body cannot be empty")
	}

	if len(msg.Body) > 4*1024*1024 { // 4MB limit
		return fmt.Errorf("message body too large: %d bytes", len(msg.Body))
	}

	return nil
}
