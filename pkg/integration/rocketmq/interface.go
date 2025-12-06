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

package rocketmq

import "context"

type TransactionState int

const (
	TransactionCommit TransactionState = iota
	TransactionRollback
	TransactionUnknown
)

type MQProducer interface {
	SendSync(ctx context.Context, msg interface{}) (interface{}, error)
	SendMessageInTransaction(ctx context.Context, msg interface{}, xid string, branchId int64) (interface{}, error)
	EndTransaction(ctx context.Context, sendResult interface{}, state TransactionState) error
}

type GlobalStatusChecker interface {
	GetGlobalStatus(ctx context.Context, xid string) (int32, error)
}
