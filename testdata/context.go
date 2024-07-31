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

package testdata

import (
	"context"

	"seata.apache.org/seata-go/pkg/protocol/message"
	"seata.apache.org/seata-go/pkg/tm"
)

const (
	TestTxName   = "TesttxName"
	TestXid      = "TestXid"
	TestXidCopy  = "TestXid"
	TestTxRole   = tm.Launcher
	TestTxStatus = message.GlobalStatusBegin
)

func GetTestContext() context.Context {
	ctx := tm.InitSeataContext(context.Background())
	tm.SetXID(ctx, TestXid)
	tm.SetTxName(ctx, TestTxName)
	tm.SetXID(ctx, TestXid)
	tm.SetXIDCopy(ctx, TestXidCopy)
	tm.SetTxRole(ctx, TestTxRole)
	tm.SetTxStatus(ctx, TestTxStatus)
	return ctx
}
