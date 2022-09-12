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

package util

import "strings"

const (
	PairSplit = "&"
	KvSplit   = "="
)

// DecodeMap Decode undo log context string to map
func DecodeMap(str string) map[string]string {
	res := make(map[string]string)

	if str == "" {
		return nil
	}

	strSlice := strings.Split(str, PairSplit)
	if len(strSlice) == 0 {
		return nil
	}

	for key, _ := range strSlice {
		kv := strings.Split(strSlice[key], KvSplit)
		if len(kv) != 2 {
			continue
		}

		res[kv[0]] = kv[1]
	}

	return res
}
