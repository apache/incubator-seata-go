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
	"github.com/seata/seata-go/pkg/tm"
	example "github.com/seata/seata-go/sample/tcc/grpc/pb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

//type BusinessClient1 struct {
//	Prepare       func(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error)          `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
//	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
//	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
//	GetActionName func() string
//}

type BusinessServer1 struct {
	example.UnimplementedTCCServiceBusiness1Server
	Prepare       func(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error)          `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	GetActionName func() string
}

//type BusinessClient2 struct {
//	Prepare       func(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error)          `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
//	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
//	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
//	GetActionName func() string
//}

type BusinessServer2 struct {
	example.UnimplementedTCCServiceBusiness1Server
	Prepare       func(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error)          `seataTwoPhaseAction:"prepare" seataTwoPhaseServiceName:"TwoPhaseDemoService"`
	Commit        func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"commit"`
	Rollback      func(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) `seataTwoPhaseAction:"rollback"`
	GetActionName func() string
}

func Prepare1(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error) {
	log.Infof("TestTCCServiceBusiness1 Prepare2, param %v", params)
	return &wrapperspb.BoolValue{Value: true}, nil
}

func Commit1(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness1 Commit2, param %v", businessActionContext)
	return true, nil
}

func Rollback1(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness1 Rollback2, param %v", businessActionContext)
	return true, nil
}

func GetActionName1() string {
	return "TCCServiceBusiness1"
}

func Prepare2(ctx context.Context, params ...interface{}) (*wrapperspb.BoolValue, error) {
	log.Infof("TestTCCServiceBusiness Prepare2, param %v", params)
	return &wrapperspb.BoolValue{Value: true}, nil
}

func Commit2(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness Commit2, param %v", businessActionContext)
	return true, nil
}

func Rollback2(ctx context.Context, businessActionContext *tm.BusinessActionContext) (bool, error) {
	log.Infof("TestTCCServiceBusiness Rollback2, param %v", businessActionContext)
	return true, nil
}

func GetActionName2() string {
	return "TCCServiceBusiness2"
}

var (
	// BusinessServer1
	BusinessServer11 = &BusinessServer1{
		Prepare:       Prepare1,
		Commit:        Commit1,
		Rollback:      Rollback1,
		GetActionName: GetActionName1,
	}

	// BusinessServer2
	BusinessServer22 = &BusinessServer2{
		Prepare:       Prepare2,
		Commit:        Commit2,
		Rollback:      Rollback2,
		GetActionName: GetActionName2,
	}
)
