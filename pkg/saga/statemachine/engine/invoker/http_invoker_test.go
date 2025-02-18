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

package invoker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/stretchr/testify/assert"
)

func TestHTTPInvokerInvokeSucceedWithOutRetry(t *testing.T) {
	// create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input []interface{}
		err := json.NewDecoder(r.Body).Decode(&input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(input[0].(string)))
	}))
	defer server.Close()

	// create HTTP Invoker
	invoker := NewHTTPInvoker()
	client := NewHTTPClient("test", server.URL+"/", &http.Client{})
	invoker.RegisterClient("test", client)

	// invoke
	ctx := context.Background()
	values, err := invoker.Invoke(ctx, []any{"hello"}, newHTTPServiceTaskState())

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, values)
	assert.Equal(t, "hello", values[0].Interface())
}

func TestHTTPInvokerInvokeWithRetry(t *testing.T) {
	attemptCount := 0
	// create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("fail"))
			return
		}
		var input []interface{}
		json.NewDecoder(r.Body).Decode(&input)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(input[0].(string)))
	}))
	defer server.Close()

	// create HTTP Invoker
	invoker := NewHTTPInvoker()
	client := NewHTTPClient("test", server.URL+"/", &http.Client{})
	invoker.RegisterClient("test", client)

	// invoker
	ctx := context.Background()
	values, err := invoker.Invoke(ctx, []any{"hello"}, newHTTPServiceTaskStateWithRetry())

	// verify
	assert.NoError(t, err)
	assert.NotNil(t, values)
	assert.Equal(t, "hello", values[0].Interface())
	assert.Equal(t, 2, attemptCount)
}

func TestHTTPInvokerInvokeFailedInRetry(t *testing.T) {
	// create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("fail"))
	}))
	defer server.Close()

	// create HTTP Invoker
	invoker := NewHTTPInvoker()
	client := NewHTTPClient("test", server.URL+"/", &http.Client{})
	invoker.RegisterClient("test", client)

	// invoker
	ctx := context.Background()
	_, err := invoker.Invoke(ctx, []any{"hello"}, newHTTPServiceTaskStateWithRetry())

	// verify
	assert.Error(t, err)
}

func TestHTTPInvokerAsyncInvoke(t *testing.T) {
	called := false
	// create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))
	defer server.Close()

	// create HTTP Invoker
	invoker := NewHTTPInvoker()
	client := NewHTTPClient("test", server.URL+"/", &http.Client{})
	invoker.RegisterClient("test", client)

	// async invoke
	ctx := context.Background()
	taskState := newHTTPServiceTaskStateWithAsync()
	_, err := invoker.Invoke(ctx, []any{"hello"}, taskState)

	// verify
	assert.NoError(t, err)
	time.Sleep(100 * time.Millisecond)
	assert.True(t, called)
}

func newHTTPServiceTaskState() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("test")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("test")
	serviceTaskStateImpl.SetServiceType("HTTP")
	serviceTaskStateImpl.SetServiceMethod("POST")
	return serviceTaskStateImpl
}

func newHTTPServiceTaskStateWithAsync() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("test")
	serviceTaskStateImpl.SetIsAsync(true)
	serviceTaskStateImpl.SetServiceName("test")
	serviceTaskStateImpl.SetServiceType("HTTP")
	serviceTaskStateImpl.SetServiceMethod("POST")
	return serviceTaskStateImpl
}

func newHTTPServiceTaskStateWithRetry() state.ServiceTaskState {
	serviceTaskStateImpl := state.NewServiceTaskStateImpl()
	serviceTaskStateImpl.SetName("test")
	serviceTaskStateImpl.SetIsAsync(false)
	serviceTaskStateImpl.SetServiceName("test")
	serviceTaskStateImpl.SetServiceType("HTTP")
	serviceTaskStateImpl.SetServiceMethod("POST")

	retryImpl := &state.RetryImpl{}
	retryImpl.SetExceptions([]string{"fail"})
	retryImpl.SetIntervalSecond(1)
	retryImpl.SetMaxAttempt(3)
	retryImpl.SetBackoffRate(0.9)
	serviceTaskStateImpl.SetRetry([]state.Retry{retryImpl})
	return serviceTaskStateImpl
}
