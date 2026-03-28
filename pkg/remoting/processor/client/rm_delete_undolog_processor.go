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
	"context"
	"database/sql"
	"fmt"
	"time"

	"seata.apache.org/seata-go/v2/pkg/datasource/sql/types"
	"seata.apache.org/seata-go/v2/pkg/datasource/sql/undo"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/remoting/getty"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

func initDeleteUndoLog() {
	getty.GetGettyClientHandlerInstance().RegisterProcessor(
		message.MessageTypeRmDeleteUndolog, &rmDeleteUndoLogProcessor{})
}

type rmDeleteUndoLogProcessor struct{}

func (r *rmDeleteUndoLogProcessor) Process(ctx context.Context, rpcMessage message.RpcMessage) error {
	req, ok := rpcMessage.Body.(message.UndoLogDeleteRequest)
	if !ok {
		return fmt.Errorf("invalid message body type: %T", rpcMessage.Body)
	}

	if req.BranchType != branch.BranchTypeAT {
		log.Infof("skip undo log delete for non-AT branch type: %v", req.BranchType)
		return nil
	}

	log.Infof("received undo log delete request: resourceId=%s, saveDays=%d",
		req.ResourceId, req.SaveDays)

	if err := r.deleteExpiredUndoLog(ctx, req); err != nil {
		log.Errorf("failed to delete expired undo log: resourceId=%s, saveDays=%d, err=%v",
			req.ResourceId, req.SaveDays, err)
	}

	return nil
}

type dbResource interface {
	GetDB() *sql.DB
	GetDbType() types.DBType
}

func (r *rmDeleteUndoLogProcessor) deleteExpiredUndoLog(ctx context.Context, req message.UndoLogDeleteRequest) error {
	resMgr, err := safeGetResourceManager(req.BranchType)
	if err != nil {
		return err
	}

	val, ok := resMgr.GetCachedResources().Load(req.ResourceId)
	if !ok {
		return fmt.Errorf("resource not found in cache: %s", req.ResourceId)
	}

	res, ok := val.(dbResource)
	if !ok {
		return fmt.Errorf("resource %s does not implement dbResource interface", req.ResourceId)
	}

	conn, err := res.GetDB().Conn(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	defer conn.Close()

	undoMgr, err := undo.GetUndoLogManager(res.GetDbType())
	if err != nil {
		return fmt.Errorf("get undo log manager for dbType %v: %w", res.GetDbType(), err)
	}

	exists, err := undoMgr.HasUndoLogTable(ctx, conn)
	if err != nil {
		return fmt.Errorf("check undo log table: %w", err)
	}
	if !exists {
		log.Infof("undo_log table not exist, skip: resourceId=%s", req.ResourceId)
		return nil
	}

	before := time.Now().AddDate(0, 0, -int(req.SaveDays))
	return r.batchDeleteByLogCreated(ctx, conn, before)
}

func (r *rmDeleteUndoLogProcessor) batchDeleteByLogCreated(ctx context.Context, conn *sql.Conn, before time.Time) error {
	undoLogTable := undo.UndoConfig.LogTable
	if undoLogTable == "" {
		undoLogTable = "undo_log"
	}

	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE log_created <= ? LIMIT 1000", undoLogTable)

	totalAffected := int64(0)
	for {
		result, err := conn.ExecContext(ctx, deleteSQL, before)
		if err != nil {
			return fmt.Errorf("exec delete: %w", err)
		}
		affected, _ := result.RowsAffected()
		totalAffected += affected
		if affected < 1000 {
			break
		}
	}

	log.Infof("deleted expired undo log: before=%v, totalAffected=%d", before, totalAffected)
	return nil
}

func safeGetResourceManager(bt branch.BranchType) (mgr rm.ResourceManager, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("resource manager not registered for branch type %v: %v", bt, r)
		}
	}()
	mgr = rm.GetRmCacheInstance().GetResourceManager(bt)
	return
}
