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

package remoting

import (
	"errors"
	"testing"
)

func TestRunSessionOpenHooks(t *testing.T) {
	var hooks []func() error
	withSessionOpenHooks(t, &hooks)
	called := 0
	RegisterSessionOpenHook(func() error {
		called++
		return nil
	})
	RegisterSessionOpenHook(nil)

	if err := RunSessionOpenHooks(); err != nil {
		t.Fatalf("RunSessionOpenHooks() error = %v", err)
	}
	if called != 1 {
		t.Fatalf("hook called %d times, want 1", called)
	}
}

func TestRunSessionOpenHooksJoinsErrorsAndContinues(t *testing.T) {
	var hooks []func() error
	withSessionOpenHooks(t, &hooks)
	firstErr := errors.New("first")
	secondErr := errors.New("second")
	called := 0
	RegisterSessionOpenHook(func() error {
		called++
		return firstErr
	})
	RegisterSessionOpenHook(func() error {
		called++
		return secondErr
	})

	err := RunSessionOpenHooks()
	if !errors.Is(err, firstErr) || !errors.Is(err, secondErr) {
		t.Fatalf("RunSessionOpenHooks() error = %v, want both hook errors", err)
	}
	if called != 2 {
		t.Fatalf("hook called %d times, want 2", called)
	}
}

func withSessionOpenHooks(t *testing.T, hooks *[]func() error) {
	t.Helper()
	sessionOpenHooksMu.Lock()
	previous := sessionOpenHooks
	sessionOpenHooks = *hooks
	sessionOpenHooksMu.Unlock()
	t.Cleanup(func() {
		sessionOpenHooksMu.Lock()
		sessionOpenHooks = previous
		sessionOpenHooksMu.Unlock()
	})
}
