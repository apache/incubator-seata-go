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

package rm

import (
	"sync"
	"testing"
)

func TestGetSagaResourceManager_Singleton(t *testing.T) {
	var wg sync.WaitGroup
	instances := make([]*SagaResourceManager, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			instances[idx] = GetSagaResourceManager()
		}(i)
	}
	wg.Wait()

	first := instances[0]
	for i, inst := range instances {
		if inst != first {
			t.Errorf("Instance at index %d is not the same as the first instance", i)
		}
	}
}
