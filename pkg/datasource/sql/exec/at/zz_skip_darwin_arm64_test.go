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

package at

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		fmt.Println("skip pkg/datasource/sql/exec/at tests on darwin/arm64 due to gomonkey instability")
		os.Exit(0)
	}
	os.Exit(m.Run())
}
