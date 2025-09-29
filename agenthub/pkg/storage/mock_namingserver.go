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

package storage

import (
	"fmt"
	"sync"

	"agenthub/pkg/utils"
)

// MockNamingserverRegistry 提供NamingserverRegistry的模拟实现，用于测试和开发
// 由于seata-go项目中的NamingServer接口方法都是TODO状态，我们使用Mock来模拟其行为
type MockNamingserverRegistry struct {
	instances     map[string][]*ServiceInstance // vGroup名称 -> 服务实例列表的映射
	currentVGroup string                        // 当前操作的vGroup上下文
	mutex         sync.RWMutex                  // 读写锁，保证并发安全
	logger        *utils.Logger                 // 日志记录器
}

// NewMockNamingserverRegistry 创建一个新的模拟NamingServer注册中心
func NewMockNamingserverRegistry() *MockNamingserverRegistry {
	return &MockNamingserverRegistry{
		instances: make(map[string][]*ServiceInstance),
		logger:    utils.WithField("component", "mock-namingserver"),
	}
}

// Register 将服务实例注册到当前vGroup上下文中
func (m *MockNamingserverRegistry) Register(instance *ServiceInstance) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 使用当前设置的vGroup，如果没有设置则使用默认值
	vGroup := m.currentVGroup
	if vGroup == "" {
		vGroup = "default-group"
	}

	m.instances[vGroup] = append(m.instances[vGroup], instance)
	m.logger.Debug("Mock: 已注册服务实例 %s:%d 到vGroup %s", instance.Addr, instance.Port, vGroup)

	return nil
}

// Deregister 从注册中心移除指定的服务实例
func (m *MockNamingserverRegistry) Deregister(instance *ServiceInstance) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// 遍历所有vGroup，查找并移除匹配的实例
	for vGroup, instances := range m.instances {
		for i, inst := range instances {
			if inst.Addr == instance.Addr && inst.Port == instance.Port {
				// 从切片中移除实例
				m.instances[vGroup] = append(instances[:i], instances[i+1:]...)
				m.logger.Debug("Mock: 已从vGroup %s 注销服务实例 %s:%d", vGroup, instance.Addr, instance.Port)
				return nil
			}
		}
	}

	return fmt.Errorf("未找到服务实例 %s:%d", instance.Addr, instance.Port)
}

// doHealthCheck 检查指定地址的服务健康状态（模拟实现）
func (m *MockNamingserverRegistry) doHealthCheck(addr string) bool {
	m.logger.Debug("Mock: 健康检查 %s - 返回健康状态", addr)
	return true // Mock实现总是返回健康状态
}

// RefreshToken 刷新指定地址的认证令牌
func (m *MockNamingserverRegistry) RefreshToken(addr string) error {
	m.logger.Debug("Mock: 刷新地址 %s 的认证令牌", addr)
	return nil // Mock实现，直接返回成功
}

// RefreshGroup 刷新指定vGroup的服务实例信息
func (m *MockNamingserverRegistry) RefreshGroup(vGroup string) error {
	m.logger.Debug("Mock: 刷新vGroup %s 的服务实例", vGroup)
	return nil // Mock实现，直接返回成功
}

// Watch 监听指定vGroup的变化
func (m *MockNamingserverRegistry) Watch(vGroup string) (bool, error) {
	m.logger.Debug("Mock: 监听vGroup %s 的变化", vGroup)
	return true, nil // Mock实现，直接返回成功
}

// Lookup 通过vGroup键查找服务实例列表
// 这是核心的服务发现方法，根据vGroup返回对应的服务实例
func (m *MockNamingserverRegistry) Lookup(key string) ([]*ServiceInstance, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	instances, exists := m.instances[key]
	if !exists {
		m.logger.Debug("Mock: vGroup %s 中未找到服务实例", key)
		return []*ServiceInstance{}, nil
	}

	// 返回副本以防止外部修改
	result := make([]*ServiceInstance, len(instances))
	copy(result, instances)

	m.logger.Debug("Mock: 在vGroup %s 中找到 %d 个服务实例", key, len(result))
	return result, nil
}

// RegisterToVGroup 将服务实例注册到指定的vGroup（测试辅助方法）
// 这个方法专门用于测试，可以直接指定vGroup而不依赖上下文
func (m *MockNamingserverRegistry) RegisterToVGroup(vGroup string, instance *ServiceInstance) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.instances[vGroup] = append(m.instances[vGroup], instance)
	m.logger.Debug("Mock: 已将服务实例 %s:%d 注册到vGroup %s", instance.Addr, instance.Port, vGroup)

	return nil
}

// GetAllGroups 返回所有已注册的vGroup列表（测试辅助方法）
func (m *MockNamingserverRegistry) GetAllGroups() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var groups []string
	for vGroup := range m.instances {
		groups = append(groups, vGroup)
	}

	return groups
}

// Clear 清除所有已注册的服务实例（测试辅助方法）
func (m *MockNamingserverRegistry) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.instances = make(map[string][]*ServiceInstance)
	m.logger.Debug("Mock: 已清除所有服务实例")
}

// SetVGroup 设置当前操作的vGroup上下文（Mock专用方法）
func (m *MockNamingserverRegistry) SetVGroup(vGroup string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.currentVGroup = vGroup
	m.logger.Debug("Mock: 设置当前vGroup为 %s", vGroup)
}

// GetCurrentVGroup 获取当前vGroup上下文（Mock专用方法）
func (m *MockNamingserverRegistry) GetCurrentVGroup() string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.currentVGroup
}
