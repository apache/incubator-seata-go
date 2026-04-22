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

package loadbalance

import (
	"sync"

	getty "github.com/apache/dubbo-getty"
)

const (
	randomLoadBalance         = "RandomLoadBalance"
	xidLoadBalance            = "XID"
	roundRobinLoadBalance     = "RoundRobinLoadBalance"
	consistentHashLoadBalance = "ConsistentHashLoadBalance"
	leastActiveLoadBalance    = "LeastActiveLoadBalance"
)

// defaultConsistent is the package-level ring used by Select so that callers
// who do not manage their own Consistent instance still get true consistent
// hashing rather than silent fallback to random.
var defaultConsistent = NewConsistent(0)

// Select dispatches to the balancer named by loadBalanceType.
// For ConsistentHashLoadBalance a package-level ring is used so that the same
// xid always maps to the same session across calls.
func Select(loadBalanceType string, sessions *sync.Map, xid string) getty.Session {
	return SelectWithConsistent(loadBalanceType, sessions, xid, defaultConsistent)
}

// SelectWithConsistent dispatches to the balancer named by loadBalanceType.
// consistent is only used by the consistent-hash strategy; other strategies
// ignore it and may safely receive nil.
func SelectWithConsistent(loadBalanceType string, sessions *sync.Map, xid string, consistent *Consistent) getty.Session {
	switch loadBalanceType {
	case randomLoadBalance:
		return RandomLoadBalance(sessions, xid)
	case xidLoadBalance:
		return XidLoadBalance(sessions, xid)
	case roundRobinLoadBalance:
		return RoundRobinLoadBalance(sessions, xid)
	case consistentHashLoadBalance:
		return ConsistentHashLoadBalance(consistent, sessions, xid)
	case leastActiveLoadBalance:
		return LeastActiveLoadBalance(sessions, xid)
	default:
		return RandomLoadBalance(sessions, xid)
	}
}
