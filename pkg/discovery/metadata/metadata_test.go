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

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRefreshMetadataGroupsNodesAndTermsByGroup(t *testing.T) {
	m := NewMetadata()

	m.RefreshMetadata("test-cluster", MetadataResponse{
		Term:      7,
		StoreMode: "RAFT",
		Nodes: []*Node{
			{
				Control:     &Endpoint{Host: "127.0.0.1", Port: 7001},
				Transaction: &Endpoint{Host: "127.0.0.1", Port: 8001},
				Internal:    &Endpoint{Host: "127.0.0.1", Port: 9001},
				Group:       "group-a",
				Role:        LEADER,
			},
			{
				Control:     &Endpoint{Host: "127.0.0.1", Port: 7002},
				Transaction: &Endpoint{Host: "127.0.0.1", Port: 8002},
				Internal:    &Endpoint{Host: "127.0.0.1", Port: 9002},
				Group:       "group-a",
				Role:        FOLLOWER,
			},
			{
				Control:     &Endpoint{Host: "127.0.0.1", Port: 7003},
				Transaction: &Endpoint{Host: "127.0.0.1", Port: 8003},
				Internal:    &Endpoint{Host: "127.0.0.1", Port: 9003},
				Group:       "group-b",
				Role:        LEADER,
			},
		},
	})

	assert.Len(t, m.GetNodes("test-cluster", "group-a"), 2)
	assert.Len(t, m.GetNodes("test-cluster", "group-b"), 1)
	assert.Equal(t, int64(7), m.GetClusterTerm("test-cluster")["group-a"])
	assert.Equal(t, int64(7), m.GetClusterTerm("test-cluster")["group-b"])
	assert.Equal(t, RAFT, m.storeMode)
}
