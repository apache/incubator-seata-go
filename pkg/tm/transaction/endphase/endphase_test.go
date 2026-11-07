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

package endphase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestErr(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()

	expired, cancelExpired := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancelExpired()

	sendErr := errors.New("connection refused")
	retriesErr := errors.New("terminated after 5 retries")

	tests := []struct {
		name       string
		ctx        context.Context
		sendErr    error
		backoffErr error
		res        interface{}
		wantNil    bool
		wantIs     error
		wantSubstr string
	}{
		{
			// The reported bug: the loop never runs, so sendErr stays nil.
			name:       "canceled before the first attempt",
			ctx:        canceled,
			backoffErr: context.Canceled,
			wantIs:     context.Canceled,
		},
		{
			name:       "deadline exceeded before the first attempt",
			ctx:        expired,
			backoffErr: context.DeadlineExceeded,
			wantIs:     context.DeadlineExceeded,
		},
		{
			name:       "canceled while retrying keeps the transport error visible",
			ctx:        canceled,
			sendErr:    sendErr,
			backoffErr: context.Canceled,
			wantIs:     context.Canceled,
			wantSubstr: "connection refused",
		},
		{
			name:       "retries exhausted",
			ctx:        context.Background(),
			sendErr:    sendErr,
			backoffErr: retriesErr,
			wantIs:     sendErr,
			wantSubstr: "terminated after 5 retries",
		},
		{
			name:    "transport error without a terminated backoff",
			ctx:     context.Background(),
			sendErr: sendErr,
			wantIs:  sendErr,
		},
		{
			name:       "backoff terminated without a transport error",
			ctx:        context.Background(),
			backoffErr: retriesErr,
			wantIs:     retriesErr,
		},
		{
			name:   "no error but no response",
			ctx:    context.Background(),
			wantIs: ErrEmptyResponse,
		},
		{
			name:    "response received",
			ctx:     context.Background(),
			res:     struct{}{},
			wantNil: true,
		},
		{
			// The transports send without a context, so a request can reach
			// the TC and come back after the deadline has passed. Discarding
			// that response would report a second phase that did complete as
			// a failure.
			name:       "response received after the context was done",
			ctx:        canceled,
			backoffErr: context.Canceled,
			res:        struct{}{},
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Err(tt.ctx, tt.sendErr, tt.backoffErr, tt.res)
			if tt.wantNil {
				if err != nil {
					t.Fatalf("Err() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Err() = nil, want an error")
			}
			if !errors.Is(err, tt.wantIs) {
				t.Fatalf("errors.Is(%v, %v) = false, want true", err, tt.wantIs)
			}
			if tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("Err() = %q, want it to mention %q", err, tt.wantSubstr)
			}
		})
	}
}

func TestUnexpectedResponse(t *testing.T) {
	err := UnexpectedResponse("global commit", 42)
	if err == nil {
		t.Fatal("UnexpectedResponse() = nil, want an error")
	}
	for _, want := range []string{"global commit", "int"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("UnexpectedResponse() = %q, want it to mention %q", err, want)
		}
	}
}
