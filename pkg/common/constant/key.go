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

package constant

const (
	GroupKey               = "group"
	VersionKey             = "version"
	InterfaceKey           = "interface"
	MessageSizeKey         = "message_size"
	PathKey                = "path"
	ServiceKey             = "service"
	MethodsKey             = "methods"
	TimeoutKey             = "timeout"
	CategoryKey            = "category"
	CheckKey               = "check"
	EnabledKey             = "enabled"
	SideKey                = "side"
	OverrideProvidersKey   = "providerAddresses"
	BeanNameKey            = "bean.name"
	GenericKey             = "generic"
	ClassifierKey          = "classifier"
	TokenKey               = "token"
	LocalAddr              = "local-addr"
	RemoteAddr             = "remote-addr"
	DefaultRemotingTimeout = 3000
	ReleaseKey             = "release"
	AnyhostKey             = "anyhost"
	PortKey                = "port"
	ProtocolKey            = "protocol"
	PathSeparator          = "/"
	DotSeparator           = "."
	CommaSeparator         = ","
	SslEnabledKey          = "ssl-enabled"
	ParamsTypeKey          = "parameter-type-names" // key used in pass through invoker factory, to define param type
	MetadataTypeKey        = "metadata-type"
	MaxCallSendMsgSize     = "max-call-send-msg-size"
	MaxServerSendMsgSize   = "max-server-send-msg-size"
	MaxCallRecvMsgSize     = "max-call-recv-msg-size"
	MaxServerRecvMsgSize   = "max-server-recv-msg-size"
)

const (
	ServerConfigPrefix                 = "seata.server"
	ServerUndoConfigPrefix             = "seata.server.undo"
	ServerRecoveryConfigPrefix         = "seata.server.recovery"
	ServerSessionConfigPrefix          = "seata.server.session"
	RegistryConfigPrefix               = "seata.registry"
	StoreConfigPrefix                  = "seata.store"
	FileConfigPrefix                   = "seata.store.file"
	DbConfigPrefix                     = "seata.store.db"
	RedisConfigPrefix                  = "seata.store.redis"
	RedisSingleConfigPrefix            = "seata.store.redis.single"
	RedisSentinelConfigPrefix          = "seata.store.redis.sentinel"
	StoreLockConfigPrefix              = "seata.store.lock"
	TransportConfigPrefix              = "seata.transport"
	TransportThreadFactoryConfigPrefix = "seata.transport.threadFactory"
	TransportShutdownConfigPrefix      = "seata.transport.shutdown"
	MetricsConfigPrefix                = "seata.metrics"
	ProfilesConfigPrefix               = "seata.profiles"
)

const (
	FileStoreMode  = "file"
	DbStoreMode    = "db"
	RedisStoreMode = "redis"
)

const (
	LocalLockMode  = "local"
	RemodeLockMode = "remote"
)
