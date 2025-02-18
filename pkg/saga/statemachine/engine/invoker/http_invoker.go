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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/util/log"
)

const errHttpCode = 400

type HTTPInvoker struct {
	clientsMapLock sync.Mutex
	clients        map[string]HTTPClient
}

func NewHTTPInvoker() *HTTPInvoker {
	return &HTTPInvoker{
		clients: make(map[string]HTTPClient),
	}
}

func (h *HTTPInvoker) RegisterClient(serviceName string, client HTTPClient) {
	h.clientsMapLock.Lock()
	defer h.clientsMapLock.Unlock()
	h.clients[serviceName] = client
}

func (h *HTTPInvoker) GetClient(serviceName string) HTTPClient {
	h.clientsMapLock.Lock()
	defer h.clientsMapLock.Unlock()
	return h.clients[serviceName]
}

func (h *HTTPInvoker) Invoke(ctx context.Context, input []any, service state.ServiceTaskState) (output []reflect.Value, err error) {
	serviceTaskStateImpl := service.(*state.ServiceTaskStateImpl)
	client := h.GetClient(serviceTaskStateImpl.ServiceName())
	if client == nil {
		return nil, fmt.Errorf("no http client %s for service task state", serviceTaskStateImpl.ServiceName())
	}

	if serviceTaskStateImpl.IsAsync() {
		go func() {
			_, err := client.Call(ctx, serviceTaskStateImpl, input)
			if err != nil {
				log.Errorf("invoke Service[%s].%s failed, err is %s", serviceTaskStateImpl.ServiceName(),
					serviceTaskStateImpl.ServiceMethod(), err)
			}
		}()
		return nil, nil
	}

	return client.Call(ctx, serviceTaskStateImpl, input)
}

func (h *HTTPInvoker) Close(ctx context.Context) error {
	return nil
}

type HTTPClient interface {
	Call(ctx context.Context, serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error)
}

type HTTPClientImpl struct {
	serviceName string
	baseURL     string
	client      *http.Client
}

func NewHTTPClient(serviceName string, baseURL string, client *http.Client) *HTTPClientImpl {
	if client == nil {
		client = &http.Client{
			Timeout: time.Second * 30,
		}
	}
	return &HTTPClientImpl{
		serviceName: serviceName,
		baseURL:     baseURL,
		client:      client,
	}
}

func (h *HTTPClientImpl) Call(ctx context.Context, serviceTaskStateImpl *state.ServiceTaskStateImpl, input []any) ([]reflect.Value, error) {
	retryCountMap := make(map[state.Retry]int)
	for {
		res, err, shouldRetry := func() (res []reflect.Value, resErr error, shouldRetry bool) {
			defer func() {
				if r := recover(); r != nil {
					errStr := fmt.Sprintf("%v", r)
					retry := h.matchRetry(serviceTaskStateImpl, errStr)
					resErr = errors.New(errStr)
					if retry != nil {
						shouldRetry = h.needRetry(serviceTaskStateImpl, retryCountMap, retry, resErr)
					}
				}
			}()

			reqBody, err := json.Marshal(input)
			if err != nil {
				return nil, err, false
			}

			req, err := http.NewRequestWithContext(ctx,
				serviceTaskStateImpl.ServiceMethod(),
				h.baseURL+serviceTaskStateImpl.Name(),
				bytes.NewBuffer(reqBody))
			if err != nil {
				return nil, err, false
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := h.client.Do(req)
			if err != nil {
				retry := h.matchRetry(serviceTaskStateImpl, err.Error())
				if retry != nil {
					return nil, err, h.needRetry(serviceTaskStateImpl, retryCountMap, retry, err)
				}
				return nil, err, false
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err, false
			}

			if resp.StatusCode >= errHttpCode {
				errStr := fmt.Sprintf("HTTP error: %d - %s", resp.StatusCode, string(body))
				retry := h.matchRetry(serviceTaskStateImpl, errStr)
				if retry != nil {
					return nil, errors.New(errStr), h.needRetry(serviceTaskStateImpl, retryCountMap, retry, err)
				}
				return nil, errors.New(errStr), false
			}

			return []reflect.Value{
				reflect.ValueOf(string(body)),
				reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()),
			}, nil, false
		}()

		if !shouldRetry {
			if err != nil {
				return nil, fmt.Errorf("invoke Service[%s] failed, not satisfy retry config, the last err is %s",
					serviceTaskStateImpl.ServiceName(), err)
			}
			return res, nil
		}
	}
}

func (h *HTTPClientImpl) matchRetry(impl *state.ServiceTaskStateImpl, str string) state.Retry {
	if impl.Retry() != nil {
		for _, retry := range impl.Retry() {
			if retry.Exceptions() != nil {
				for _, exception := range retry.Exceptions() {
					if strings.Contains(str, exception) {
						return retry
					}
				}
			}
		}
	}
	return nil
}

func (h *HTTPClientImpl) needRetry(impl *state.ServiceTaskStateImpl, countMap map[state.Retry]int, retry state.Retry, err error) bool {
	attempt, exist := countMap[retry]
	if !exist {
		countMap[retry] = 0
	}

	if attempt >= retry.MaxAttempt() {
		return false
	}

	intervalSecond := retry.IntervalSecond()
	backoffRate := retry.BackoffRate()
	var currentInterval int64
	if attempt == 0 {
		currentInterval = int64(intervalSecond * 1000)
	} else {
		currentInterval = int64(intervalSecond * backoffRate * float64(attempt) * 1000)
	}

	log.Warnf("invoke service[%s.%s] failed, will retry after %s millis, current retry count: %s, current err: %s",
		impl.ServiceName(), impl.ServiceMethod(), currentInterval, attempt, err)

	time.Sleep(time.Duration(currentInterval) * time.Millisecond)
	countMap[retry] = attempt + 1
	return true
}
