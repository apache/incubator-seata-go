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

package service

import (
	"context"
	"github.com/seata/seata-go/pkg/common/log"
	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/sample/tcc/grpc/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Business struct {
	pb.UnimplementedTCCServiceBusinessServer
}

// Remoting is your rpc method be defined in proto IDL, you must use TccServiceProxy to proxy your business Object in rpc method , e.g. the Remoting method
func (b *Business) Remoting(ctx context.Context, params *pb.Params) (*emptypb.Empty, error) {
	log.Infof("Remoting be called")
	return new(emptypb.Empty), tcc.NewTCCServiceProxy(b).Prepare(ctx, params)
}

func (b *Business) Prepare(ctx context.Context, params interface{}) error {
	log.Infof("TestTCCServiceBusiness Prepare, param %v", params)
	return nil
}

func (b *Business) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Commit, param %v", businessActionContext)
	return nil
}

func (b *Business) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	log.Infof("TestTCCServiceBusiness Rollback, param %v", businessActionContext)
	return nil
}

func (b *Business) GetActionName() string {
	return "TCCServiceBusiness"
}
