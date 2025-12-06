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

package collection

import (
	"container/list"
	"strings"
)

const (
	KvSplit   = "="
	PairSplit = "&"
)

var (
	kvSplitBytes   = []byte(KvSplit)
	pairSplitBytes = []byte(PairSplit)
)

func EncodeMap(dataMap map[string]string) []byte {
	if dataMap == nil {
		return nil
	}

	bytes := make([]byte, 0)
	if len(dataMap) == 0 {
		return bytes
	}

	for k, v := range dataMap {
		bytes = append(bytes, []byte(k)...)
		bytes = append(bytes, kvSplitBytes...)
		bytes = append(bytes, []byte(v)...)
		bytes = append(bytes, pairSplitBytes...)
	}

	return bytes[:len(bytes)-1]
}

func DecodeMap(data []byte) map[string]string {
	if data == nil {
		return nil
	}

	ctxMap := make(map[string]string, 0)

	dataStr := string(data)
	if dataStr == "" {
		return ctxMap
	}

	kvPairs := strings.Split(dataStr, PairSplit)
	if len(kvPairs) == 0 {
		return ctxMap
	}

	for _, kvPair := range kvPairs {
		if kvPair == "" {
			continue
		}

		kvs := strings.Split(kvPair, KvSplit)
		if len(kvs) != 2 {
			continue
		}

		ctxMap[kvs[0]] = kvs[1]
	}

	return ctxMap
}

type Stack struct {
	list *list.List
}

func NewStack() *Stack {
	list := list.New()
	return &Stack{list}
}

func (stack *Stack) Push(value interface{}) {
	stack.list.PushBack(value)
}

func (stack *Stack) Pop() interface{} {
	e := stack.list.Back()
	if e != nil {
		stack.list.Remove(e)
		return e.Value
	}
	return nil
}

func (stack *Stack) Peak() interface{} {
	e := stack.list.Back()
	if e != nil {
		return e.Value
	}

	return nil
}

func (stack *Stack) Len() int {
	return stack.list.Len()
}

func (stack *Stack) Empty() bool {
	return stack.list.Len() == 0
}
