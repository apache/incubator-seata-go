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

package remoting

import (
	"reflect"
)

type RemoteType byte
type ServiceType byte

const (
	RemoteTypeSofaRpc      RemoteType = 2
	RemoteTypeDubbo        RemoteType = 3
	RemoteTypeRestful      RemoteType = 4
	RemoteTypeLocalService RemoteType = 5
	RemoteTypeHsf          RemoteType = 8

	ServiceTypeReference ServiceType = 1
	ServiceTypeProvider  ServiceType = 2
)

type RemotingDesc struct {
	/**
	 * is referenc bean ?
	 */
	isReference bool

	/**
	 * rpc target bean, the service bean has this property
	 */
	targetBean interface{}

	/**
	 * the tcc interface tyep
	 */
	interfaceClass reflect.Type

	/**
	 * interface class name
	 */
	interfaceClassName string

	/**
	 * rpc uniqueId: hsf, dubbo's version, sofa-rpc's uniqueId
	 */
	uniqueId string

	/**
	 * dubbo/hsf 's group
	 */
	group string

	/**
	 * protocol: sofa-rpc, dubbo, injvm etc.
	 */
	protocol RemoteType
}

type RemotingParser interface {
	isRemoting(bean interface{}, beanName string) (bool, error)

	/**
	 * if it is gprc bean ?
	 */
	isReference(bean interface{}, beanName string) (bool, error)

	/**
	 * if it is service bean ?
	 */
	isService(bean interface{}, beanName string) (bool, error)

	/**
	 * get the remoting bean info
	 */
	getServiceDesc(bean interface{}, beanName string) (RemotingDesc, error)

	/**
	 * the remoting protocol
	 */
	getProtocol() RemoteType
}
