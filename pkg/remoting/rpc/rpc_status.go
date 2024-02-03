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

package rpc

import (
	"sync"
	"sync/atomic"
)

var serviceStatusMap sync.Map

type Status struct {
	Active int32
	Total  int32
}

// RemoveStatus remove the RpcStatus of this service
func RemoveStatus(service string) {
	serviceStatusMap.Delete(service)
}

// BeginCount begin count
func BeginCount(service string) {
	status := GetStatus(service)
	atomic.AddInt32(&status.Active, 1)
}

// EndCount end count
func EndCount(service string) {
	status := GetStatus(service)
	atomic.AddInt32(&status.Active, -1)
	atomic.AddInt32(&status.Total, 1)
}

// GetStatus get status
func GetStatus(service string) *Status {
	a, _ := serviceStatusMap.LoadOrStore(service, new(Status))
	return a.(*Status)
}

// GetActive get active.
func (s *Status) GetActive() int32 {
	return s.Active
}

// GetTotal get total.
func (s *Status) GetTotal() int32 {
	return s.Total
}
