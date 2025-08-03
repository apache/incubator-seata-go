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

package discovery

type NamingserverRegistry interface {
	RegistryService

	Register(instance *ServiceInstance) error

	Deregister(instance *ServiceInstance) error

	// doHealthCheck
	// perform a health check and call the/amine/v1/health interface
	doHealthCheck(addr string) bool

	// RefreshToken
	// Refresh the JWT token and call the /api/v1/auth/login interface.
	RefreshToken(addr string) error

	// RefreshGroup
	// Refresh the service group information and call the /naming/v1/discovery interface.
	RefreshGroup(vGroup string) error

	// Watch
	// Monitor service changes and call the /naming/v1/watch interface.
	Watch(vGroup string) (bool, error)
}
