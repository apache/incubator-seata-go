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

package db

import (
	"database/sql"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"regexp"
	"time"
)

const (
	StateMachineFields                   = "id, tenant_id, app_name, name, status, gmt_create, ver, type, content, recover_strategy, comment_"
	GetStateMachineByIdSql               = "SELECT " + StateMachineFields + " FROM ${TABLE_PREFIX}state_machine_def WHERE id = ?"
	QueryStateMachinesByNameAndTenantSql = "SELECT " + StateMachineFields + " FROM ${TABLE_PREFIX}state_machine_def WHERE name = ? AND tenant_id = ? ORDER BY gmt_create DESC"
	InsertStateMachineSql                = "INSERT INTO ${TABLE_PREFIX}state_machine_def (" + StateMachineFields + ") VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	TablePrefix                          = "\\$\\{TABLE_PREFIX}"
)

type StateLangStore struct {
	Store
	tablePrefix                          string
	getStateMachineByIdSql               string
	queryStateMachinesByNameAndTenantSql string
	insertStateMachineSql                string
}

func NewStateLangStore(db *sql.DB, tablePrefix string) *StateLangStore {
	r := regexp.MustCompile(TablePrefix)

	stateLangStore := &StateLangStore{
		Store:                                Store{db},
		tablePrefix:                          tablePrefix,
		getStateMachineByIdSql:               r.ReplaceAllString(GetStateMachineByIdSql, tablePrefix),
		queryStateMachinesByNameAndTenantSql: r.ReplaceAllString(QueryStateMachinesByNameAndTenantSql, tablePrefix),
		insertStateMachineSql:                r.ReplaceAllString(InsertStateMachineSql, tablePrefix),
	}

	return stateLangStore
}

func (s *StateLangStore) GetStateMachineById(stateMachineId string) (statelang.StateMachine, error) {
	return SelectOne(s.db, s.getStateMachineByIdSql, scanRowsToStateMachine, stateMachineId)
}

func (s *StateLangStore) GetLastVersionStateMachine(stateMachineName string, tenantId string) (statelang.StateMachine, error) {
	stateMachineList, err := SelectList(s.db, s.queryStateMachinesByNameAndTenantSql, scanRowsToStateMachine, stateMachineName, tenantId)
	if err != nil {
		return nil, err
	}

	if len(stateMachineList) > 0 {
		return stateMachineList[0], nil
	}
	return nil, nil
}

func (s *StateLangStore) StoreStateMachine(stateMachine statelang.StateMachine) error {
	rows, err := ExecuteUpdate(s.db, s.insertStateMachineSql, execStateMachineStatement, stateMachine)
	if err != nil {
		return err
	}
	if rows <= 0 {
		return errors.New("affected rows is smaller than 0")
	}

	return nil
}

func scanRowsToStateMachine(rows *sql.Rows) (statelang.StateMachine, error) {
	stateMachine := statelang.NewStateMachineImpl()
	//var id, name, comment, version, appName, content, t, recoverStrategy, tenantId, status string
	var id, tenantId, appName, name, status, created, version, t, content, recoverStrategy, comment string
	//var created int64
	err := rows.Scan(&id, &tenantId, &appName, &name, &status, &created, &version, &t, &content, &recoverStrategy, &comment)
	if err != nil {
		return stateMachine, err
	}
	stateMachine.SetID(id)
	stateMachine.SetName(name)
	stateMachine.SetComment(comment)
	stateMachine.SetVersion(version)
	stateMachine.SetAppName(appName)
	stateMachine.SetContent(content)
	createdTime, _ := time.Parse(TimeLayout, created)
	stateMachine.SetCreateTime(createdTime)
	stateMachine.SetType(t)
	if recoverStrategy != "" {
		stateMachine.SetRecoverStrategy(statelang.RecoverStrategy(recoverStrategy))
	}
	stateMachine.SetTenantId(tenantId)
	stateMachine.SetStatus(statelang.StateMachineStatus(status))
	return stateMachine, nil
}

func execStateMachineStatement(obj statelang.StateMachine, stmt *sql.Stmt) (int64, error) {
	result, err := stmt.Exec(
		obj.ID(),
		obj.TenantId(),
		obj.AppName(),
		obj.Name(),
		obj.Status(),
		obj.CreateTime(),
		obj.Version(),
		obj.Type(),
		obj.Content(),
		obj.RecoverStrategy(),
		obj.Comment(),
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}
