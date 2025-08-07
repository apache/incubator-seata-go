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
	"sync"
	"testing"
	"time"

	"github.com/robertkrimen/otto"
	"github.com/stretchr/testify/assert"
)

func TestJavaScriptScriptInvoker_Type(t *testing.T) {
	invoker := NewJavaScriptScriptInvoker()
	assert.Equal(t, "javascript", invoker.Type())
}

func TestJavaScriptScriptInvoker_Invoke_Basic(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		params   map[string]interface{}
		expected interface{}
	}{
		{
			name:     "simple expression",
			script:   "1 + 2",
			params:   nil,
			expected: float64(3),
		},
		{
			name:     "param calculation",
			script:   "a * b + c",
			params:   map[string]interface{}{"a": 2, "b": 3, "c": 4},
			expected: float64(10),
		},
		{
			name:     "return string",
			script:   "['hello', name].join(' ')",
			params:   map[string]interface{}{"name": "world"},
			expected: "hello world",
		},
		{
			name:     "return object",
			script:   `var obj = {id: 1, name: name}; obj;`,
			params:   map[string]interface{}{"name": "test"},
			expected: map[string]interface{}{"id": float64(1), "name": "test"},
		},
	}

	invoker := NewJavaScriptScriptInvoker()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := invoker.Invoke(ctx, tt.script, tt.params)
			assert.NoError(t, err)

			if resultMap, ok := result.(map[string]interface{}); ok {
				for k, v := range resultMap {
					if intVal, isInt := v.(int64); isInt {
						resultMap[k] = float64(intVal)
					}
				}
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJavaScriptScriptInvoker_Invoke_Error(t *testing.T) {
	tests := []struct {
		name   string
		script string
		params map[string]interface{}
		errMsg string
	}{
		{
			name:   "syntax error",
			script: "1 + ",
			params: nil,
			errMsg: "javascript execute error",
		},
		{
			name:   "reference undefined variable",
			script: "undefinedVar",
			params: nil,
			errMsg: "javascript execute error",
		},
	}

	invoker := NewJavaScriptScriptInvoker()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := invoker.Invoke(ctx, tt.script, tt.params)

			if err == nil {
				t.Fatalf("Test case [%s] expected error but got none", tt.name)
			}
			assert.Contains(t, err.Error(), tt.errMsg, "Test case [%s] error message mismatch", tt.name)
		})
	}
}

func TestJavaScriptScriptInvoker_Invoke_Timeout(t *testing.T) {

	script := `var target = 300; var start = new Date().getTime(); var elapsed = 0; while (elapsed < target) { elapsed = new Date().getTime() - start; } "done";`
	invoker := NewJavaScriptScriptInvoker()

	ctx1, cancel1 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel1()
	_, err := invoker.Invoke(ctx1, script, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "javascript execution timeout")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel2()
	result, err := invoker.Invoke(ctx2, script, nil)
	assert.NoError(t, err, "Scenario 2: script execution should not return error")
	assert.Equal(t, "done", result, "Scenario 2: should return 'done'")
}

func TestJavaScriptScriptInvoker_Invoke_Concurrent(t *testing.T) {
	invoker := NewJavaScriptScriptInvoker()
	ctx := context.Background()
	var wg sync.WaitGroup
	concurrency := 100
	errChan := make(chan error, concurrency)

	script := `a + b`
	params := map[string]interface{}{"a": 10, "b": 20}

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := invoker.Invoke(ctx, script, params)
			if err != nil {
				errChan <- err
				return
			}
			if result != float64(30) {
				errChan <- assert.AnError
			}
		}()
	}

	wg.Wait()
	close(errChan)

	assert.Empty(t, errChan, "Concurrent execution has errors")
}

func TestJavaScriptScriptInvoker_Close(t *testing.T) {
	invoker := NewJavaScriptScriptInvoker()
	ctx := context.Background()

	result, err := invoker.Invoke(ctx, "1 + 1", nil)
	assert.NoError(t, err)
	assert.Equal(t, float64(2), result)

	err = invoker.Close(ctx)
	assert.NoError(t, err)

	_, err = invoker.Invoke(ctx, "1 + 1", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "javascript invoker has been closed")
}

func TestOttoScript(t *testing.T) {
	vm := otto.New()
	script := `var target = 300; var start = new Date().getTime(); var elapsed = 0; while (elapsed < target) { elapsed = new Date().getTime() - start; } "done";`
	val, err := vm.Run(script)
	if err != nil {
		t.Fatalf("otto failed to parse script: %v", err)
	}
	
	result, exportErr := val.Export()
	if exportErr != nil {
		t.Fatalf("failed to export otto value: %v", exportErr)
	}
	t.Logf("Script execution result: %v", result)
}
