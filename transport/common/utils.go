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

package common

import (
	"encoding/json"
	"fmt"
)

// MarshalMessage marshals a message to JSON
func MarshalMessage(v interface{}) (*Message, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	return &Message{
		Payload: data,
		Headers: make(map[string]string),
	}, nil
}

// UnmarshalResponse unmarshals a response from JSON
func UnmarshalResponse(resp *Response, v interface{}) error {
	if resp.Error != nil {
		return resp.Error
	}

	if err := json.Unmarshal(resp.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// UnmarshalStreamResponse unmarshals a stream response from JSON
func UnmarshalStreamResponse(resp *StreamResponse, v interface{}) error {
	if resp.Error != nil {
		return resp.Error
	}

	if resp.Done {
		return nil
	}

	if err := json.Unmarshal(resp.Data, v); err != nil {
		return fmt.Errorf("failed to unmarshal stream response: %w", err)
	}

	return nil
}

// CreateResponse creates a response with JSON marshaled data
func CreateResponse(v interface{}) (*Response, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return &Response{
		Data:    data,
		Headers: make(map[string]string),
	}, nil
}

// CreateStreamResponse creates a stream response with JSON marshaled data
func CreateStreamResponse(v interface{}, done bool) (*StreamResponse, error) {
	var data []byte
	var err error

	if v != nil && !done {
		data, err = json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal stream response: %w", err)
		}
	}

	return &StreamResponse{
		Data:    data,
		Headers: make(map[string]string),
		Done:    done,
	}, nil
}

// CreateErrorResponse creates an error response
func CreateErrorResponse(err error) *Response {
	return &Response{
		Data:    nil,
		Headers: make(map[string]string),
		Error:   err,
	}
}

// CreateErrorStreamResponse creates an error stream response
func CreateErrorStreamResponse(err error) *StreamResponse {
	return &StreamResponse{
		Data:    nil,
		Headers: make(map[string]string),
		Done:    true,
		Error:   err,
	}
}

// SetHeader sets a header in message
func (m *Message) SetHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
}

// GetHeader gets a header from message
func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// SetHeader sets a header in response
func (r *Response) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
}

// GetHeader gets a header from response
func (r *Response) GetHeader(key string) string {
	if r.Headers == nil {
		return ""
	}
	return r.Headers[key]
}

// SetHeader sets a header in stream response
func (sr *StreamResponse) SetHeader(key, value string) {
	if sr.Headers == nil {
		sr.Headers = make(map[string]string)
	}
	sr.Headers[key] = value
}

// GetHeader gets a header from stream response
func (sr *StreamResponse) GetHeader(key string) string {
	if sr.Headers == nil {
		return ""
	}
	return sr.Headers[key]
}
