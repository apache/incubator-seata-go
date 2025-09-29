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

package jsonrpc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string `json:"id,omitempty"`
	Event string `json:"event,omitempty"`
	Data  string `json:"data"`
	Retry int    `json:"retry,omitempty"`
}

// String formats the SSE event for transmission
func (e *SSEEvent) String() string {
	var sb strings.Builder

	if e.ID != "" {
		sb.WriteString(fmt.Sprintf("id: %s\n", e.ID))
	}
	if e.Event != "" {
		sb.WriteString(fmt.Sprintf("event: %s\n", e.Event))
	}
	if e.Retry > 0 {
		sb.WriteString(fmt.Sprintf("retry: %d\n", e.Retry))
	}

	// Handle multi-line data
	lines := strings.Split(e.Data, "\n")
	for _, line := range lines {
		sb.WriteString(fmt.Sprintf("data: %s\n", line))
	}

	sb.WriteString("\n")
	return sb.String()
}

// SSEParser parses Server-Sent Events from a reader
type SSEParser struct {
	scanner *bufio.Scanner
	ctx     context.Context
}

// NewSSEParser creates a new SSE parser
func NewSSEParser(ctx context.Context, reader io.Reader) *SSEParser {
	return &SSEParser{
		scanner: bufio.NewScanner(reader),
		ctx:     ctx,
	}
}

// NextEvent reads the next SSE event
func (p *SSEParser) NextEvent() (*SSEEvent, error) {
	event := &SSEEvent{}
	var dataLines []string

	for {
		select {
		case <-p.ctx.Done():
			return nil, p.ctx.Err()
		default:
		}

		if !p.scanner.Scan() {
			if err := p.scanner.Err(); err != nil {
				return nil, fmt.Errorf("scanner error: %w", err)
			}
			// EOF
			if len(dataLines) > 0 || event.ID != "" || event.Event != "" {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			return nil, io.EOF
		}

		line := p.scanner.Text()

		// Empty line indicates end of event
		if line == "" {
			if len(dataLines) > 0 || event.ID != "" || event.Event != "" {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			continue
		}

		// Skip comments
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Parse field
		colonIndex := strings.Index(line, ":")
		if colonIndex == -1 {
			// Field name only
			switch line {
			case "data":
				dataLines = append(dataLines, "")
			}
			continue
		}

		field := line[:colonIndex]
		value := line[colonIndex+1:]

		// Remove leading space from value
		if len(value) > 0 && value[0] == ' ' {
			value = value[1:]
		}

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "retry":
			// Parse retry value (not implemented in this basic version)
		}
	}
}
