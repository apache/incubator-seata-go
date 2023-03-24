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

package registry

import (
	"net"
)

// RegistryService defined registry service
type RegistryService interface {

	// RegisterServiceInstance register new service to nacos
	RegisterServiceInstance(address net.TCPAddr)

	// DeRegisterServiceInstance deRegister new service to nacos
	DeRegisterServiceInstance(address net.TCPAddr)

	// Subscribe key=serviceName+groupName+cluster
	// Note:We call add multiple SubscribeCallback with the same key.
	Subscribe(cluster string, groupName string)

	// UnSubscribe key=serviceName+groupName+cluster
	UnSubscribe(cluster string, groupName string)
}
