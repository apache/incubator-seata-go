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

package config

import "github.com/seata/seata-go/pkg/common/constant"

type StoreConfig struct {
	Mode             string               `default:"file",yaml:"mode"  json:"mode,omitempty" property:"mode"`
	StoreModeLock    *StoreModeLockConfig `yaml:"lock"  json:"lock,omitempty" property:"lock"`
	FileStoreConfig  *FileStoreConfig     `yaml:"file" json:"file,omitempty",property:"file"`
	DBStoreConfig    *DbStoreConfig       `yaml:"db" json:"db,omitempty",property:"db"`
	RedisStoreConfig *RedisStoreConfig    `yaml:"redis" json:"redis,omitempty",property:"redis"`
}

type FileStoreConfig struct {
	Dir                      string `default:"sessionStore" yaml:"dir" json:"dir,omitempty",property:"dir"`
	MaxBranchSessionSize     int    `default:"16384" yaml:"max-branch-session-size" json:"maxBranchSessionSize,omitempty",property:"maxBranchSessionSize"`
	MaxGlobalSessionSize     int    `default:"512" yaml:"max-global-session-size" json:"maxGlobalSessionSize,omitempty",property:"maxGlobalSessionSize"`
	FileWriteBufferCacheSize int    `default:"16384" yaml:"file-write-buffer-cache-size" json:"fileWriteBufferCacheSize,omitempty",property:"fileWriteBufferCacheSize"`
	FlushDiskMode            string `default:"async" yaml:"flush-disk-mode" json:"flushDiskMode,omitempty",property:"flushDiskMode"`
	SessionReloadReadSize    int    `default:"100" yaml:"session-reload-read-size" json:"sessionReloadReadSize,omitempty",property:"sessionReloadReadSize"`
}

type DbStoreConfig struct {
	Datasource           string `default:"druid" yaml:"datasource" json:"datasource,omitempty",property:"datasource"`
	DbType               string `default:"mysql" yaml:"db-type" json:"dbType,omitempty",property:"dbType"`
	DriverClassName      string `default:"com.mysql.jdbc.Driver" yaml:"driver-class-name" json:"driverClassName,omitempty",property:"driverClassName"`
	Url                  string `default:"jdbc:mysql://127.0.0.1:3306/seata?rewriteBatchedStatements=true" yaml:"url" json:"url,omitempty",property:"url"`
	User                 string `default:"mysql" yaml:"user" json:"user,omitempty",property:"user"`
	Password             string `default:"mysql" yaml:"password" json:"password,omitempty",property:"password"`
	MinConn              int    `default:"5" yaml:"min-conn" json:"minConn,omitempty",property:"minConn"`
	MaxConn              int    `default:"100" yaml:"max-conn" json:"maxConn,omitempty",property:"maxConn"`
	GlobalTable          string `default:"global_table" yaml:"global-table" json:"globalTable,omitempty",property:"globalTable"`
	BranchTable          string `default:"branch_table" yaml:"branch-table" json:"branchTable,omitempty",property:"branchTable"`
	LockTable            string `default:"lock_table" yaml:"lock-table" json:"lockTable,omitempty",property:"lockTable"`
	DistributedLockTable string `default:"distributed_lock" yaml:"distributed-lock-table" json:"distributedLockTable,omitempty",property:"distributedLockTable"`
	QueryLimit           int    `default:"100" yaml:"query-limit" json:"queryLimit,omitempty",property:"queryLimit"`
	MaxWait              int    `default:"5000" yaml:"max-wait" json:"maxWait,omitempty",property:"maxWait"`
}

type StoreModeLockConfig struct {
	Mode string `default:"file" yaml:"mode" json:"mode,omitempty",property:"mode"`
}

type RedisStoreConfig struct {
	Mode           string          `default:"single" yaml:"mode" json:"mode,omitempty",property:"mode"`
	Database       int             `default:"0" yaml:"database" json:"database,omitempty",property:"database"`
	MinConn        int             `default:"1" yaml:"min-conn" json:"minConn,omitempty",property:"minConn"`
	MaxConn        int             `default:"10" yaml:"max-conn" json:"maxConn,omitempty",property:"maxConn"`
	Password       string          `default:"mysql" yaml:"password" json:"password,omitempty",property:"password"`
	MaxTotal       int             `default:"100" yaml:"max-total" json:"maxtotal,omitempty",property:"maxtotal"`
	QueryLimit     int             `default:"100" yaml:"query-limit" json:"queryLimit,omitempty",property:"queryLimit"`
	SingleConf     *SingleConfig   `yaml:"single" json:"single,omitempty",property:"single"`
	SentinelConfig *SentinelConfig `yaml:"sentinel" json:"sentinel,omitempty",property:"sentinel"`
}

type SingleConfig struct {
	Host string `default:"127.0.0.1" yaml:"host" json:"host,omitempty",property:"host"`
	Port int    `default:"6379" yaml:"port" json:"port,omitempty",property:"port"`
}

type SentinelConfig struct {
	MasterName    string `yaml:"master-name" json:"masterName,omitempty",property:"masterName"`
	SentinelHosts string `yaml:"sentinel-hosts" json:"sentinelHosts,omitempty",property:"sentinelHosts"`
}

// Prefix seata.registries
func (StoreConfig) Prefix() string {
	return constant.StoreConfigPrefix
}

// Prefix seata.store
func (StoreModeLockConfig) Prefix() string {
	return constant.StoreLockConfigPrefix
}

// Prefix seata.store.file
func (FileStoreConfig) Prefix() string {
	return constant.FileConfigPrefix
}

// Prefix seata.store.db
func (DbStoreConfig) Prefix() string {
	return constant.DbConfigPrefix
}

// Prefix seata.store.redis
func (RedisStoreConfig) Prefix() string {
	return constant.RedisConfigPrefix
}

// Prefix seata.store.redis.single
func (SingleConfig) Prefix() string {
	return constant.RedisSingleConfigPrefix
}

// Prefix seata.store.redis.sentinel
func (SentinelConfig) Prefix() string {
	return constant.RedisSentinelConfigPrefix
}
