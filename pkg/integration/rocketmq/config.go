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
	"time"

	"github.com/apache/rocketmq-client-go/v2/producer"
)

type SeataMQProducerConfig struct {
	NameServerAddrs []string
	GroupName       string
	Namespace       string
	InstanceName    string

	RetryTimesWhenSendFailed int
	SendMsgTimeout           time.Duration
}

func NewDefaultSeataMQProducerConfig() *SeataMQProducerConfig {
	return &SeataMQProducerConfig{
		RetryTimesWhenSendFailed: 3,
		SendMsgTimeout:           3 * time.Second,
	}
}

func (c *SeataMQProducerConfig) ToRocketMQProducerOptions() []producer.Option {
	opts := []producer.Option{
		producer.WithNameServer(c.NameServerAddrs),
		producer.WithGroupName(c.GroupName),
		producer.WithRetry(c.RetryTimesWhenSendFailed),
		producer.WithSendMsgTimeout(c.SendMsgTimeout),
	}
	if c.Namespace != "" {
		opts = append(opts, producer.WithNamespace(c.Namespace))
	}
	if c.InstanceName != "" {
		opts = append(opts, producer.WithInstanceName(c.InstanceName))
	}
	return opts
}
