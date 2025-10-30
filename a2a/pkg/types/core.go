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
	"encoding/json"
	"fmt"
	"time"

	pb "seata-go-ai-a2a/pkg/proto/v1"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Task is the core unit of action for A2A protocol
type Task struct {
	ID        string         `json:"id"`
	ContextID string         `json:"contextId"`
	Status    *TaskStatus    `json:"status"`
	Artifacts []*Artifact    `json:"artifacts"`
	History   []*Message     `json:"history"`
	Kind      string         `json:"kind"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// TaskStatus contains the status information for a task
type TaskStatus struct {
	State     TaskState `json:"state"`
	Update    *Message  `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Message represents one unit of communication between client and server
type Message struct {
	MessageID  string         `json:"messageId"`
	ContextID  string         `json:"contextId,omitempty"`
	TaskID     string         `json:"taskId,omitempty"`
	Role       Role           `json:"role"`
	Parts      []Part         `json:"parts"`
	Kind       string         `json:"kind"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Extensions []string       `json:"extensions,omitempty"`
}

// Artifact represents the container for task completed results
type Artifact struct {
	ArtifactID  string         `json:"artifactId"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parts       []Part         `json:"parts"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Extensions  []string       `json:"extensions,omitempty"`
}

// Part represents a container for a section of communication content
type Part interface {
	PartType() PartTypeEnum
}

// PartTypeEnum identifies the type of a Part
type PartTypeEnum int

const (
	PartTypeText PartTypeEnum = iota
	PartTypeFile
	PartTypeData
)

// TextPart represents plain text content
type TextPart struct {
	Text     string         `json:"text"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (t *TextPart) PartType() PartTypeEnum { return PartTypeText }

// FilePart represents file content
type FilePart struct {
	Content  FileContent    `json:"content"`
	MimeType string         `json:"mime_type"`
	Name     string         `json:"name"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (f *FilePart) PartType() PartTypeEnum { return PartTypeFile }

// FileContent represents different ways files can be provided
type FileContent interface {
	FileContentType() FileContentTypeEnum
}

// FileContentTypeEnum identifies the type of file content
type FileContentTypeEnum int

const (
	FileContentTypeURI FileContentTypeEnum = iota
	FileContentTypeBytes
)

// FileWithURI represents a file reference by URI
type FileWithURI struct {
	URI string `json:"uri"`
}

func (f *FileWithURI) FileContentType() FileContentTypeEnum { return FileContentTypeURI }

// FileWithBytes represents file content as bytes
type FileWithBytes struct {
	Bytes []byte `json:"bytes"`
}

func (f *FileWithBytes) FileContentType() FileContentTypeEnum { return FileContentTypeBytes }

// DataPart represents structured data content
type DataPart struct {
	Data     map[string]any `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (d *DataPart) PartType() PartTypeEnum { return PartTypeData }

// TaskToProto converts Task to pb.Task
func TaskToProto(task *Task) (*pb.Task, error) {
	if task == nil {
		return nil, nil
	}

	var status *pb.TaskStatus
	var err error
	if task.Status != nil {
		status, err = TaskStatusToProto(task.Status)
		if err != nil {
			return nil, fmt.Errorf("converting task status: %w", err)
		}
	}

	artifacts := make([]*pb.Artifact, len(task.Artifacts))
	for i, artifact := range task.Artifacts {
		artifacts[i], err = ArtifactToProto(artifact)
		if err != nil {
			return nil, fmt.Errorf("converting artifact %d: %w", i, err)
		}
	}

	history := make([]*pb.Message, len(task.History))
	for i, msg := range task.History {
		history[i], err = MessageToProto(msg)
		if err != nil {
			return nil, fmt.Errorf("converting history message %d: %w", i, err)
		}
	}

	metadata, err := MapToStruct(task.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting task metadata: %w", err)
	}

	return &pb.Task{
		Id:        task.ID,
		ContextId: task.ContextID,
		Status:    status,
		Artifacts: artifacts,
		History:   history,
		Kind:      task.Kind,
		Metadata:  metadata,
	}, nil
}

// TaskFromProto converts pb.Task to Task
func TaskFromProto(task *pb.Task) (*Task, error) {
	if task == nil {
		return nil, nil
	}

	var status *TaskStatus
	var err error
	if task.Status != nil {
		status, err = TaskStatusFromProto(task.Status)
		if err != nil {
			return nil, fmt.Errorf("converting task status: %w", err)
		}
	}

	artifacts := make([]*Artifact, len(task.Artifacts))
	for i, artifact := range task.Artifacts {
		artifacts[i], err = ArtifactFromProto(artifact)
		if err != nil {
			return nil, fmt.Errorf("converting artifact %d: %w", i, err)
		}
	}

	history := make([]*Message, len(task.History))
	for i, msg := range task.History {
		history[i], err = MessageFromProto(msg)
		if err != nil {
			return nil, fmt.Errorf("converting history message %d: %w", i, err)
		}
	}

	metadata, err := StructToMap(task.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting task metadata: %w", err)
	}

	return &Task{
		ID:        task.Id,
		ContextID: task.ContextId,
		Status:    status,
		Artifacts: artifacts,
		History:   history,
		Kind:      task.Kind,
		Metadata:  metadata,
	}, nil
}

// TaskStatusToProto converts TaskStatus to pb.TaskStatus
func TaskStatusToProto(status *TaskStatus) (*pb.TaskStatus, error) {
	if status == nil {
		return nil, nil
	}

	var update *pb.Message
	var err error
	if status.Update != nil {
		update, err = MessageToProto(status.Update)
		if err != nil {
			return nil, fmt.Errorf("converting status update: %w", err)
		}
	}

	return &pb.TaskStatus{
		State:     TaskStateToProto(status.State),
		Update:    update,
		Timestamp: timestamppb.New(status.Timestamp),
	}, nil
}

// TaskStatusFromProto converts pb.TaskStatus to TaskStatus
func TaskStatusFromProto(status *pb.TaskStatus) (*TaskStatus, error) {
	if status == nil {
		return nil, nil
	}

	var update *Message
	var err error
	if status.Update != nil {
		update, err = MessageFromProto(status.Update)
		if err != nil {
			return nil, fmt.Errorf("converting status update: %w", err)
		}
	}

	return &TaskStatus{
		State:     TaskStateFromProto(status.State),
		Update:    update,
		Timestamp: status.Timestamp.AsTime(),
	}, nil
}

// MessageToProto converts Message to pb.Message
func MessageToProto(msg *Message) (*pb.Message, error) {
	if msg == nil {
		return nil, nil
	}

	content := make([]*pb.Part, len(msg.Parts))
	for i, part := range msg.Parts {
		var err error
		content[i], err = PartToProto(part)
		if err != nil {
			return nil, fmt.Errorf("converting content part %d: %w", i, err)
		}
	}

	metadata, err := MapToStruct(msg.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting message metadata: %w", err)
	}

	return &pb.Message{
		MessageId:  msg.MessageID,
		ContextId:  msg.ContextID,
		TaskId:     msg.TaskID,
		Role:       RoleToProto(msg.Role),
		Content:    content,
		Kind:       msg.Kind,
		Metadata:   metadata,
		Extensions: msg.Extensions,
	}, nil
}

// MessageFromProto converts pb.Message to Message
func MessageFromProto(msg *pb.Message) (*Message, error) {
	if msg == nil {
		return nil, nil
	}

	content := make([]Part, len(msg.Content))
	for i, part := range msg.Content {
		var err error
		content[i], err = PartFromProto(part)
		if err != nil {
			return nil, fmt.Errorf("converting content part %d: %w", i, err)
		}
	}

	metadata, err := StructToMap(msg.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting message metadata: %w", err)
	}

	return &Message{
		MessageID:  msg.MessageId,
		ContextID:  msg.ContextId,
		TaskID:     msg.TaskId,
		Role:       RoleFromProto(msg.Role),
		Parts:      content,
		Kind:       msg.Kind,
		Metadata:   metadata,
		Extensions: msg.Extensions,
	}, nil
}

// ArtifactToProto converts Artifact to pb.Artifact
func ArtifactToProto(artifact *Artifact) (*pb.Artifact, error) {
	if artifact == nil {
		return nil, nil
	}

	parts := make([]*pb.Part, len(artifact.Parts))
	for i, part := range artifact.Parts {
		var err error
		parts[i], err = PartToProto(part)
		if err != nil {
			return nil, fmt.Errorf("converting artifact part %d: %w", i, err)
		}
	}

	metadata, err := MapToStruct(artifact.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting artifact metadata: %w", err)
	}

	return &pb.Artifact{
		ArtifactId:  artifact.ArtifactID,
		Name:        artifact.Name,
		Description: artifact.Description,
		Parts:       parts,
		Metadata:    metadata,
		Extensions:  artifact.Extensions,
	}, nil
}

// ArtifactFromProto converts pb.Artifact to Artifact
func ArtifactFromProto(artifact *pb.Artifact) (*Artifact, error) {
	if artifact == nil {
		return nil, nil
	}

	parts := make([]Part, len(artifact.Parts))
	for i, part := range artifact.Parts {
		var err error
		parts[i], err = PartFromProto(part)
		if err != nil {
			return nil, fmt.Errorf("converting artifact part %d: %w", i, err)
		}
	}

	metadata, err := StructToMap(artifact.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting artifact metadata: %w", err)
	}

	return &Artifact{
		ArtifactID:  artifact.ArtifactId,
		Name:        artifact.Name,
		Description: artifact.Description,
		Parts:       parts,
		Metadata:    metadata,
		Extensions:  artifact.Extensions,
	}, nil
}

// PartToProto converts Part to pb.Part
func PartToProto(part Part) (*pb.Part, error) {
	if part == nil {
		return nil, nil
	}

	pbPart := &pb.Part{}

	switch p := part.(type) {
	case *TextPart:
		pbPart.Part = &pb.Part_Text{Text: p.Text}
		if len(p.Metadata) > 0 {
			metadata, err := MapToStruct(p.Metadata)
			if err != nil {
				return nil, fmt.Errorf("converting text part metadata: %w", err)
			}
			pbPart.Metadata = metadata
		}
	case *FilePart:
		filePart, err := FilePartToProto(p)
		if err != nil {
			return nil, fmt.Errorf("converting file part: %w", err)
		}
		pbPart.Part = &pb.Part_File{File: filePart}
		if len(p.Metadata) > 0 {
			metadata, err := MapToStruct(p.Metadata)
			if err != nil {
				return nil, fmt.Errorf("converting file part metadata: %w", err)
			}
			pbPart.Metadata = metadata
		}
	case *DataPart:
		dataPart, err := DataPartToProto(p)
		if err != nil {
			return nil, fmt.Errorf("converting data part: %w", err)
		}
		pbPart.Part = &pb.Part_Data{Data: dataPart}
		if len(p.Metadata) > 0 {
			metadata, err := MapToStruct(p.Metadata)
			if err != nil {
				return nil, fmt.Errorf("converting data part metadata: %w", err)
			}
			pbPart.Metadata = metadata
		}
	default:
		return nil, fmt.Errorf("unknown part type: %T", part)
	}

	return pbPart, nil
}

// PartFromProto converts pb.Part to Part
func PartFromProto(part *pb.Part) (Part, error) {
	if part == nil {
		return nil, nil
	}

	metadata, err := StructToMap(part.Metadata)
	if err != nil {
		return nil, fmt.Errorf("converting part metadata: %w", err)
	}

	switch p := part.Part.(type) {
	case *pb.Part_Text:
		return &TextPart{
			Text:     p.Text,
			Metadata: metadata,
		}, nil
	case *pb.Part_File:
		filePart, err := FilePartFromProto(p.File)
		if err != nil {
			return nil, fmt.Errorf("converting file part: %w", err)
		}
		filePart.Metadata = metadata
		return filePart, nil
	case *pb.Part_Data:
		dataPart, err := DataPartFromProto(p.Data)
		if err != nil {
			return nil, fmt.Errorf("converting data part: %w", err)
		}
		dataPart.Metadata = metadata
		return dataPart, nil
	default:
		return nil, fmt.Errorf("unknown part type: %T", p)
	}
}

// FilePartToProto converts FilePart to pb.FilePart
func FilePartToProto(filePart *FilePart) (*pb.FilePart, error) {
	if filePart == nil {
		return nil, nil
	}

	pbFilePart := &pb.FilePart{
		MimeType: filePart.MimeType,
		Name:     filePart.Name,
	}

	switch content := filePart.Content.(type) {
	case *FileWithURI:
		pbFilePart.File = &pb.FilePart_FileWithUri{FileWithUri: content.URI}
	case *FileWithBytes:
		pbFilePart.File = &pb.FilePart_FileWithBytes{FileWithBytes: content.Bytes}
	default:
		return nil, fmt.Errorf("unknown file content type: %T", content)
	}

	return pbFilePart, nil
}

// FilePartFromProto converts pb.FilePart to FilePart
func FilePartFromProto(filePart *pb.FilePart) (*FilePart, error) {
	if filePart == nil {
		return nil, nil
	}

	result := &FilePart{
		MimeType: filePart.MimeType,
		Name:     filePart.Name,
	}

	switch file := filePart.File.(type) {
	case *pb.FilePart_FileWithUri:
		result.Content = &FileWithURI{URI: file.FileWithUri}
	case *pb.FilePart_FileWithBytes:
		result.Content = &FileWithBytes{Bytes: file.FileWithBytes}
	default:
		return nil, fmt.Errorf("unknown file type: %T", file)
	}

	return result, nil
}

// DataPartToProto converts DataPart to pb.DataPart
func DataPartToProto(dataPart *DataPart) (*pb.DataPart, error) {
	if dataPart == nil {
		return nil, nil
	}

	data, err := MapToStruct(dataPart.Data)
	if err != nil {
		return nil, fmt.Errorf("converting data part data: %w", err)
	}

	return &pb.DataPart{
		Data: data,
	}, nil
}

// DataPartFromProto converts pb.DataPart to DataPart
func DataPartFromProto(dataPart *pb.DataPart) (*DataPart, error) {
	if dataPart == nil {
		return nil, nil
	}

	data, err := StructToMap(dataPart.Data)
	if err != nil {
		return nil, fmt.Errorf("converting data part data: %w", err)
	}

	return &DataPart{
		Data: data,
	}, nil
}

// MapToStruct converts a map[string]any to structpb.Struct
func MapToStruct(m map[string]any) (*structpb.Struct, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return structpb.NewStruct(m)
}

// StructToMap converts a structpb.Struct to map[string]any
func StructToMap(s *structpb.Struct) (map[string]any, error) {
	if s == nil {
		return nil, nil
	}
	return s.AsMap(), nil
}

// PartWrapper wraps Part interface for JSON serialization
type PartWrapper struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// MarshalPart marshals a Part to JSON with type information
func MarshalPart(part Part) ([]byte, error) {
	if part == nil {
		return []byte("null"), nil
	}

	var wrapper PartWrapper
	switch part.PartType() {
	case PartTypeText:
		wrapper = PartWrapper{Type: "text", Data: part}
	case PartTypeFile:
		wrapper = PartWrapper{Type: "file", Data: part}
	case PartTypeData:
		wrapper = PartWrapper{Type: "data", Data: part}
	default:
		return nil, fmt.Errorf("unknown part type: %v", part.PartType())
	}

	return json.Marshal(wrapper)
}

// UnmarshalPart unmarshals JSON to a Part with type information
func UnmarshalPart(data []byte) (Part, error) {
	if string(data) == "null" {
		return nil, nil
	}

	var wrapper PartWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to unmarshal part wrapper: %w", err)
	}

	dataBytes, err := json.Marshal(wrapper.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal part data: %w", err)
	}

	switch wrapper.Type {
	case "text":
		var textPart TextPart
		if err := json.Unmarshal(dataBytes, &textPart); err != nil {
			return nil, fmt.Errorf("failed to unmarshal text part: %w", err)
		}
		return &textPart, nil
	case "file":
		var filePart FilePart
		if err := json.Unmarshal(dataBytes, &filePart); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file part: %w", err)
		}
		return &filePart, nil
	case "data":
		var dataPart DataPart
		if err := json.Unmarshal(dataBytes, &dataPart); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data part: %w", err)
		}
		return &dataPart, nil
	default:
		return nil, fmt.Errorf("unknown part type: %s", wrapper.Type)
	}
}

// MessageJSON is a helper type for JSON serialization of Message
type MessageJSON struct {
	MessageID  string            `json:"messageId"`
	ContextID  string            `json:"contextId,omitempty"`
	TaskID     string            `json:"taskId,omitempty"`
	Role       Role              `json:"role"`
	Parts      []json.RawMessage `json:"parts"`
	Kind       string            `json:"kind"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
	Extensions []string          `json:"extensions,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Message
func (m *Message) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}

	// Convert parts to JSON
	parts := make([]json.RawMessage, len(m.Parts))
	for i, part := range m.Parts {
		partBytes, err := MarshalPart(part)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal part %d: %w", i, err)
		}
		parts[i] = json.RawMessage(partBytes)
	}

	msgJSON := MessageJSON{
		MessageID:  m.MessageID,
		ContextID:  m.ContextID,
		TaskID:     m.TaskID,
		Role:       m.Role,
		Parts:      parts,
		Kind:       m.Kind,
		Metadata:   m.Metadata,
		Extensions: m.Extensions,
	}

	return json.Marshal(msgJSON)
}

// UnmarshalJSON implements custom JSON unmarshaling for Message
func (m *Message) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var msgJSON MessageJSON
	if err := json.Unmarshal(data, &msgJSON); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Convert parts from JSON
	parts := make([]Part, len(msgJSON.Parts))
	for i, partData := range msgJSON.Parts {
		part, err := UnmarshalPart([]byte(partData))
		if err != nil {
			return fmt.Errorf("failed to unmarshal part %d: %w", i, err)
		}
		parts[i] = part
	}

	m.MessageID = msgJSON.MessageID
	m.ContextID = msgJSON.ContextID
	m.TaskID = msgJSON.TaskID
	m.Role = msgJSON.Role
	m.Parts = parts
	m.Kind = msgJSON.Kind
	m.Metadata = msgJSON.Metadata
	m.Extensions = msgJSON.Extensions

	return nil
}

// ArtifactJSON is a helper type for JSON serialization of Artifact
type ArtifactJSON struct {
	ArtifactID  string            `json:"artifactId"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Parts       []json.RawMessage `json:"parts"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Extensions  []string          `json:"extensions,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Artifact
func (a *Artifact) MarshalJSON() ([]byte, error) {
	if a == nil {
		return []byte("null"), nil
	}

	// Convert parts to JSON
	parts := make([]json.RawMessage, len(a.Parts))
	for i, part := range a.Parts {
		partBytes, err := MarshalPart(part)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal part %d: %w", i, err)
		}
		parts[i] = json.RawMessage(partBytes)
	}

	artifactJSON := ArtifactJSON{
		ArtifactID:  a.ArtifactID,
		Name:        a.Name,
		Description: a.Description,
		Parts:       parts,
		Metadata:    a.Metadata,
		Extensions:  a.Extensions,
	}

	return json.Marshal(artifactJSON)
}

// UnmarshalJSON implements custom JSON unmarshaling for Artifact
func (a *Artifact) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		return nil
	}

	var artifactJSON ArtifactJSON
	if err := json.Unmarshal(data, &artifactJSON); err != nil {
		return fmt.Errorf("failed to unmarshal artifact: %w", err)
	}

	// Convert parts from JSON
	parts := make([]Part, len(artifactJSON.Parts))
	for i, partData := range artifactJSON.Parts {
		part, err := UnmarshalPart([]byte(partData))
		if err != nil {
			return fmt.Errorf("failed to unmarshal part %d: %w", i, err)
		}
		parts[i] = part
	}

	a.ArtifactID = artifactJSON.ArtifactID
	a.Name = artifactJSON.Name
	a.Description = artifactJSON.Description
	a.Parts = parts
	a.Metadata = artifactJSON.Metadata
	a.Extensions = artifactJSON.Extensions

	return nil
}
