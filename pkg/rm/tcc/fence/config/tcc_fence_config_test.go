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

package config

import (
	"testing"

	"seata.apache.org/seata-go/pkg/util/log"
)

func TestInitFence(t *testing.T) {
	log.Init()

	// InitFence is currently empty, just call it to ensure no panic
	InitFence()
}

func TestInitCleanTask(t *testing.T) {
	log.Init()

	// Test with an invalid DSN - should log warning but not panic
	dsn := "invalid-dsn"

	// Call InitCleanTask which starts a goroutine
	// Even with invalid DSN, it should handle the error gracefully
	InitCleanTask(dsn)

	// Note: We don't call Destroy() here because there's a known issue
	// where Destroy() panics if the logQueue was not initialized
	// (which happens when the DSN is invalid and sql.Open fails).
	// This is a limitation of the current handler implementation.
}
