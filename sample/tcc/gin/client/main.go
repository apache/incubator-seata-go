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
	"context"
	"flag"
	"time"

	"github.com/parnurzeal/gorequest"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

func main() {
	flag.Parse()
	client.Init()
	bgCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	serverIpPort := "http://127.0.0.1:8080"

	tm.WithGlobalTx(
		bgCtx,
		&tm.GtxConfig{
			Name: "TccSampleLocalGlobalTx",
		},
		func(ctx context.Context) (re error) {
			request := gorequest.New()
			log.Infof("branch transaction begin")
			request.Post(serverIpPort+"/prepare").
				Set(constant.XidKey, tm.GetXID(ctx)).
				End(func(response gorequest.Response, body string, errs []error) {
					if len(errs) != 0 {
						re = errs[0]
					}
				})
			return
		})
}
