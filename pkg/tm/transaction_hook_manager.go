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

package tm

import (
	"sync"

	"github.com/pkg/errors"
)

var (
	transactionalHookManager     *TransactionalHookManager
	onceTransactionalHookManager sync.Once
)

type TransactionalHookManager struct {
	transactionalHook []TransactionalHook
}

func GetTransactionalHookManager() *TransactionalHookManager {
	if transactionalHookManager == nil {
		onceTransactionalHookManager.Do(func() {
			transactionalHookManager = &TransactionalHookManager{
				transactionalHook: make([]TransactionalHook, 0),
			}
		})
	}

	return transactionalHookManager
}

func (h *TransactionalHookManager) GetHooks() []TransactionalHook {
	return h.transactionalHook
}

func (h *TransactionalHookManager) RegisterHook(hook TransactionalHook) error {
	if nil == hook {
		return errors.New("transactionHook must not be null")
	}
	h.transactionalHook = append(h.transactionalHook, hook)
	return nil
}

func (h *TransactionalHookManager) Clear() {
	h.transactionalHook = h.transactionalHook[0:0]
}
