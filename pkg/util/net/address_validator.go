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

package net

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	addressSplitChar = ":"
)

func SplitIPPortStr(addr string) (string, int, error) {
	if addr == "" {
		return "", 0, fmt.Errorf("split ip err: param addr must not empty")
	}

	if addr[0] == '[' {
		reg := regexp.MustCompile("[\\[\\]]")
		addr = reg.ReplaceAllString(addr, "")
	}

	i := strings.LastIndex(addr, addressSplitChar)
	if i < 0 {
		return "", 0, fmt.Errorf("address %s: missing port in address", addr)
	}

	host := addr[:i]
	port := addr[i+1:]

	if strings.Contains(host, "%") {
		reg := regexp.MustCompile("\\%[0-9]+")
		host = reg.ReplaceAllString(host, "")
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	return host, portInt, nil
}
