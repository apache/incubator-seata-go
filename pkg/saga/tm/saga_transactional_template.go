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

package tm

import (
	"context"
	"time"

	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/tm"
)

type SagaTransactionalTemplate interface {
	CommitTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error

	RollbackTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error

	BeginTransaction(ctx context.Context, timeout time.Duration, txName string) (*tm.GlobalTransaction, error)

	ReloadTransaction(ctx context.Context, xid string) (*tm.GlobalTransaction, error)

	ReportTransaction(ctx context.Context, gtr *tm.GlobalTransaction) error

	BranchRegister(ctx context.Context, resourceId string, clientId string, xid string, applicationData string, lockKeys string) error

	BranchReport(ctx context.Context, xid string, branchId int64, status branch.BranchStatus, applicationData string) error

	CleanUp(ctx context.Context)
}
