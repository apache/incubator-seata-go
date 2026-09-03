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

// Package endphase normalizes the outcome of a global transaction second
// phase so that every transport classifies the same situation the same way.
package endphase

import (
	"context"
	"errors"
	"fmt"
)

// ErrEmptyResponse reports a second phase that finished without a transport
// error but produced no response for the caller to read a status from.
var ErrEmptyResponse = errors.New("empty second phase response")

// Err turns the outcome of a commit or rollback retry loop into a single error
// value, and returns nil only when res holds a response the caller can use.
//
// The retry loops stop as soon as the backoff is no longer Ongoing, which also
// happens when the caller's context is already done before the first attempt.
// No request is sent in that case and sendErr stays nil, so callers must never
// treat a nil sendErr on its own as success.
//
// Checks run in a fixed order:
//
//  1. a response that came back, which outranks everything else
//  2. the caller's context, so a canceled or expired context is reported as
//     such and stays matchable with errors.Is
//  3. the last transport error, annotated with why the retries stopped
//  4. the backoff terminating on its own, that is, retries exhausted
//  5. a missing response
func Err(ctx context.Context, sendErr, backoffErr error, res interface{}) error {
	// A response outranks a done context. The transports send without a
	// context, so the retry loop can outlive the deadline while the request
	// itself still reached the TC and came back. Reporting a failure here
	// would hide a second phase that actually completed.
	if sendErr == nil && res != nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		if sendErr != nil {
			return fmt.Errorf("%w, last transport error: %v", ctxErr, sendErr)
		}
		return ctxErr
	}
	if sendErr != nil {
		if backoffErr != nil {
			return fmt.Errorf("%w, %v", sendErr, backoffErr)
		}
		return sendErr
	}
	if backoffErr != nil {
		return backoffErr
	}
	return ErrEmptyResponse
}

// UnexpectedResponse reports a response whose type does not match the one the
// request expects. It replaces the bare type assertions that used to panic on
// a nil or mistyped response.
func UnexpectedResponse(action string, res interface{}) error {
	return fmt.Errorf("unexpected %s response type %T", action, res)
}

// IncompleteResponse reports a response of the expected type that is missing
// the payload holding the global status. Protobuf getters return zero values
// for a missing message, so without this check the transaction would silently
// take on status zero.
func IncompleteResponse(action string) error {
	return fmt.Errorf("incomplete %s response", action)
}
