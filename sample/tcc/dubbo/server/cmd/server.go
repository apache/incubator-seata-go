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

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/sample/tcc/dubbo/server/service"
)

// need to setup environment variable "DUBBO_GO_CONFIG_PATH" to "conf/dubbogo.yml" before run
func main() {
	client.InitPath("./sample/conf/seatago.yml")
	userProviderProxy, err := tcc.NewTCCServiceProxy(&service.UserProvider{})
	if err != nil {
		log.Errorf("get userProviderProxy tcc service proxy error, %v", err.Error())
		return
	}
	config.SetProviderService(userProviderProxy)
	if err := config.Load(); err != nil {
		panic(err)
	}
	initSignal()
}

func initSignal() {
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		log.Infof("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
			// reload()
		default:
			time.AfterFunc(time.Duration(int(3e9)), func() {
				log.Warnf("app exit now by force...")
				os.Exit(1)
			})
			// The program exits normally or timeout forcibly exits.
			fmt.Println("provider app exit now...")
			return
		}
	}
}
