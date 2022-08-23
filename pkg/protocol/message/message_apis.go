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

package message

type AbstractResultMessage struct {
	ResultCode ResultCode
	Msg        string
}

type AbstractIdentifyRequest struct {
	Version                 string
	ApplicationId           string `json:"applicationId"`
	TransactionServiceGroup string
	ExtraData               []byte
}

type AbstractIdentifyResponse struct {
	AbstractResultMessage
	Version    string
	ExtraData  []byte
	Identified bool
}

type MessageTypeAware interface {
	GetTypeCode() MessageType
}

type MergedWarpMessage struct {
	Msgs   []MessageTypeAware
	MsgIds []int32
}

func (req MergedWarpMessage) GetTypeCode() MessageType {
	return MessageTypeSeataMerge
}

type MergeResultMessage struct {
	Msgs []MessageTypeAware
}

func (resp MergeResultMessage) GetTypeCode() MessageType {
	return MessageTypeSeataMergeResult
}

type ResultCode byte

const (
	ResultCodeFailed  = ResultCode(0)
	ResultCodeSuccess = ResultCode(1)
)
