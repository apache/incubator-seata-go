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
	"fmt"
	"sync"

	"github.com/dop251/goja"
)

type JavaScriptScriptInvoker struct {
	mutex      sync.Mutex
	jsonParser JsonParser
	closed     bool
}

func NewJavaScriptScriptInvoker() *JavaScriptScriptInvoker {
	return &JavaScriptScriptInvoker{
		jsonParser: &DefaultJsonParser{},
		closed:     false,
	}
}

func (j *JavaScriptScriptInvoker) Type() string {
	return "javascript"
}

func (j *JavaScriptScriptInvoker) Invoke(ctx context.Context, script string, params map[string]interface{}) (interface{}, error) {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.closed {
		return nil, fmt.Errorf("javascript invoker has been closed")
	}

	vm := goja.New()

	for key, value := range params {
		if err := vm.Set(key, value); err != nil {
			return nil, fmt.Errorf("javascript set param %s error: %w", key, err)
		}
	}

	resultChan := make(chan struct {
		val goja.Value
		err error
	}, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- struct {
					val goja.Value
					err error
				}{nil, fmt.Errorf("javascript engine panic: %v", r)}
			}
		}()

		val, err := vm.RunString(script)
		resultChan <- struct {
			val goja.Value
			err error
		}{val, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("javascript execution timeout: %w", ctx.Err())
	case res := <-resultChan:
		if res.err != nil {
			return nil, fmt.Errorf("javascript execute error: %w", res.err)
		}
		return res.val.Export(), nil
	}
}

func (j *JavaScriptScriptInvoker) Close(ctx context.Context) error {
	j.mutex.Lock()
	defer j.mutex.Unlock()

	if j.closed {
		return nil
	}

	j.closed = true
	return nil
}
