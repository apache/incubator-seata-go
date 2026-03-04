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
	"math/rand"
	"sync"
)

type MetadataResponse struct {
	Nodes     []*Node
	StoreMode string
	Term      int64
}

type Metadata struct {
	leaders      sync.Map // clusterName -> sync.Map(group -> *Node)
	clusterTerm  sync.Map // clusterName -> sync.Map(group -> int64)
	clusterNodes sync.Map // clusterName -> sync.Map(group -> []*Node)
	storeMode    StoreMode
}

func NewMetadata() *Metadata {
	return &Metadata{
		storeMode: FILE,
	}
}

func (m *Metadata) GetLeader(clusterName string) *Node {
	if groupMapAny, ok := m.leaders.Load(clusterName); ok {
		if groupMap, ok := groupMapAny.(*sync.Map); ok {
			var nodes []*Node
			groupMap.Range(func(_, value any) bool {
				if node, ok := value.(*Node); ok {
					nodes = append(nodes, node)
				}
				return true
			})
			if len(nodes) > 0 {
				return nodes[rand.Intn(len(nodes))]
			}
		}
	}
	return nil
}

func (m *Metadata) GetNodes(clusterName, group string) []*Node {
	clusterNodesAny, ok := m.clusterNodes.Load(clusterName)
	if !ok {
		return nil
	}
	clusterMap, ok := clusterNodesAny.(*sync.Map)
	if !ok {
		return nil
	}

	if group == "" {
		var result []*Node
		clusterMap.Range(func(_, value any) bool {
			if nodes, ok := value.([]*Node); ok {
				result = append(result, nodes...)
			}
			return true
		})
		return result
	}

	if nodesAny, ok := clusterMap.Load(group); ok {
		if nodes, ok := nodesAny.([]*Node); ok {
			return nodes
		}
	}
	return nil
}

func (m *Metadata) SetNodes(clusterName, group string, nodes []*Node) {
	clusterMapAny, _ := m.clusterNodes.LoadOrStore(clusterName, &sync.Map{})
	clusterMap := clusterMapAny.(*sync.Map)
	clusterMap.Store(group, nodes)
}

func (m *Metadata) ContainsGroup(clusterName string) bool {
	_, ok := m.clusterNodes.Load(clusterName)
	return ok
}

func (m *Metadata) Groups(clusterName string) []string {
	if clusterAny, ok := m.clusterNodes.Load(clusterName); ok {
		if cluster, ok := clusterAny.(*sync.Map); ok {
			var groups []string
			cluster.Range(func(key, _ any) bool {
				if group, ok := key.(string); ok {
					groups = append(groups, group)
				}
				return true
			})
			return groups
		}
	}
	return nil
}

func (m *Metadata) GetClusterTerm(clusterName string) map[string]int64 {
	if termMapAny, ok := m.clusterTerm.Load(clusterName); ok {
		if termMap, ok := termMapAny.(*sync.Map); ok {
			result := make(map[string]int64)
			termMap.Range(func(key, value any) bool {
				k, ok1 := key.(string)
				v, ok2 := value.(int64)
				if ok1 && ok2 {
					result[k] = v
				}
				return true
			})
			return result
		}
	}
	return nil
}

func (m *Metadata) RefreshMetadata(clusterName string, response MetadataResponse) {
	var nodes []*Node
	for _, node := range response.Nodes {
		if node.Role == LEADER {
			groupMapAny, _ := m.leaders.LoadOrStore(clusterName, &sync.Map{})
			groupMap := groupMapAny.(*sync.Map)
			groupMap.Store(node.Group, node)
		}
		nodes = append(nodes, node)
	}

	switch response.StoreMode {
	case "RAFT":
		m.storeMode = RAFT
	default:
		m.storeMode = FILE
	}

	if len(nodes) > 0 {
		group := nodes[0].Group
		m.SetNodes(clusterName, group, nodes)
		termMapAny, _ := m.clusterTerm.LoadOrStore(clusterName, &sync.Map{})
		termMap := termMapAny.(*sync.Map)
		termMap.Store(group, response.Term)
	}
}
