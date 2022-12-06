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
	"errors"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/parnurzeal/gorequest"

	"github.com/seata/seata-go/pkg/client"
	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
)

var serverIpPort = "http://127.0.0.1:8080"

func main() {
	flag.Parse()
	client.Init()

	bgCtx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	transInfo := &tm.GtxConfig{
		Name:    "ATSampleLocalGlobalTx",
		Timeout: time.Second * 30,
	}

	if err := tm.WithGlobalTx(bgCtx, transInfo, updateData); err != nil {
		panic(fmt.Sprintf("tm update data err, %v", err))
	}
}

func updateData(ctx context.Context) (re error) {
	request := gorequest.New()
	log.Infof("branch transaction begin")

	request.Post(serverIpPort+"/updateDataSuccess").
		Set(constant.XidKey, tm.GetXID(ctx)).
		End(func(response gorequest.Response, body string, errs []error) {
			if response.StatusCode != http.StatusOK {
				re = errors.New("update data fail")
			}
		})
	return
}
