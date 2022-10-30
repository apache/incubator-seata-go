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

	"github.com/seata/seata-go/pkg/rm/tcc"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
	"github.com/seata/seata-go/sample/tcc/grpc/pb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type GrpcBusinessService1 struct {
	pb.UnimplementedTCCServiceBusiness1Server
	Business1 *tcc.TCCServiceProxy
}

type Business1 struct{}

// Remoting is your rpc method be defined in proto IDL, you must use TccServiceProxy to proxy your business Object in rpc method , e.g. the Remoting method
func (b *GrpcBusinessService1) Remoting(ctx context.Context, params *pb.Params) (*wrapperspb.BoolValue, error) {
	log.Infof("Remoting be called")
	res, err := b.Business1.Prepare(ctx, params)
	if err != nil {
		return wrapperspb.Bool(false), err
	}
	return wrapperspb.Bool(res.(bool)), nil
}

func (b *Business1) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("TestTCCServiceBusiness1 Prepare, param %v", params)
	return true, nil
}

func (b *Business1) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness1 Commit, param %v", businessActionContext)
	return true, nil
}

func (b *Business1) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness1 Rollback, param %v", businessActionContext)
	return true, nil
}

func (b *Business1) GetActionName() string {
	return "TCCServiceBusiness1"
}

type GrpcBusinessService2 struct {
	pb.UnimplementedTCCServiceBusiness2Server
	Business2 *tcc.TCCServiceProxy
}

type Business2 struct{}

// Remoting is your rpc method be defined in proto IDL, you must use TccServiceProxy to proxy your business Object in rpc method , e.g. the Remoting method
func (b *GrpcBusinessService2) Remoting(ctx context.Context, params *pb.Params) (*anypb.Any, error) {
	log.Infof("Remoting be called")
	anyFalse, err := anypb.New(wrapperspb.Bool(false))
	if err != nil {
		return nil, err
	}

	res, err := b.Business2.Prepare(ctx, params)
	if err != nil {
		return anyFalse, err
	}
	AnyBool, err := anypb.New(wrapperspb.Bool(res.(bool)))
	if err != nil {
		return nil, err
	}
	return AnyBool, nil
}

func (b *Business2) Prepare(ctx context.Context, params interface{}) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Prepare, param %v", params)
	return true, nil
}

func (b *Business2) Commit(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Commit, param %v", businessActionContext)
	return true, nil
}

func (b *Business2) Rollback(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness2 Rollback, param %v", businessActionContext)
	return true, nil
}

func (b *Business2) GetActionName() string {
	return "TCCServiceBusiness2"
}
