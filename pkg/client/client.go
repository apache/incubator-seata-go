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

package client

import (
	"sync"

	_ "github.com/seata/seata-go/pkg/integration"
	"github.com/seata/seata-go/pkg/remoting/getty"
	_ "github.com/seata/seata-go/pkg/remoting/processor/client"
	_ "github.com/seata/seata-go/pkg/rm/tcc"
)

// Init seata client client
func Init() {
	InitPath("")
}

// Init init client client with config path
func InitPath(configFilePath string) {
	cfg := LoadPath(configFilePath)

	initRmClient(cfg)
	initTmClient(cfg)
}

var onceInitTmClient sync.Once

// InitTmClient init client tm client
func initTmClient(cfg *Config) {
	onceInitTmClient.Do(func() {
		initRemoting(cfg)
	})
}

// initRemoting init rpc client
func initRemoting(cfg *Config) {
	getty.InitRpcClient()
}

// InitRmClient init client rm client
func initRmClient(cfg *Config) {
}
