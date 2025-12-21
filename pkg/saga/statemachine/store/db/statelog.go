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
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	constant2 "seata.apache.org/seata-go/v2/pkg/constant"
	"seata.apache.org/seata-go/v2/pkg/protocol/branch"
	"seata.apache.org/seata-go/v2/pkg/protocol/message"
	"seata.apache.org/seata-go/v2/pkg/rm"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine"
	engExc "seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/exception"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/pcext"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/sequence"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/engine/serializer"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/process_ctrl"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/v2/pkg/saga/statemachine/statelang/state"
	sagaTm "seata.apache.org/seata-go/v2/pkg/saga/tm"
	"seata.apache.org/seata-go/v2/pkg/tm"
	seataErrors "seata.apache.org/seata-go/v2/pkg/util/errors"
	"seata.apache.org/seata-go/v2/pkg/util/log"
)

const (
	StateMachineInstanceFields              = "id, machine_id, tenant_id, parent_id, business_key, gmt_started, gmt_end, status, compensation_status, is_running, gmt_updated, start_params, end_params, excep"
	StateMachineInstanceFieldsWithoutParams = "id, machine_id, tenant_id, parent_id, business_key, gmt_started, gmt_end, status, compensation_status, is_running, gmt_updated"
	RecordStateMachineStartedSql            = "INSERT INTO ${TABLE_PREFIX}state_machine_inst\n(id, machine_id, tenant_id, parent_id, gmt_started, business_key, start_params, is_running, status,  gmt_updated)\n VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	RecordStateMachineFinishedSql           = "UPDATE ${TABLE_PREFIX}state_machine_inst SET gmt_end = ?, excep = ?, end_params = ?,status = ?, compensation_status = ?, is_running = ?, gmt_updated = ? WHERE id = ? and gmt_updated = ?"
	UpdateStateMachineRunningStatusSql      = "UPDATE ${TABLE_PREFIX}state_machine_inst SET\nis_running = ?, gmt_updated = ? where id = ? and gmt_updated = ?"
	GetStateMachineInstanceByIdSql          = "SELECT " + StateMachineInstanceFields + " FROM ${TABLE_PREFIX}state_machine_inst WHERE id = ?"
	GetStateMachineInstanceByBusinessKeySql = "SELECT " + StateMachineInstanceFields + " FROM ${TABLE_PREFIX}state_machine_inst WHERE business_key = ? AND tenant_id = ?"
	QueryStateMachineInstancesByParentIdSql = "SELECT " + StateMachineInstanceFieldsWithoutParams + " FROM ${TABLE_PREFIX}state_machine_inst WHERE parent_id = ? ORDER BY gmt_started DESC"

	StateInstanceFields                         = "id, machine_inst_id, name, type, business_key, gmt_started, service_name, service_method, service_type, is_for_update, status, input_params, output_params, excep, gmt_end, state_id_compensated_for, state_id_retried_for"
	RecordStateStartedSql                       = "INSERT INTO ${TABLE_PREFIX}state_inst (id, machine_inst_id, name, type, gmt_started, service_name, service_method, service_type, is_for_update, input_params, status, business_key, state_id_compensated_for, state_id_retried_for, gmt_updated)\nVALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
	RecordStateFinishedSql                      = "UPDATE ${TABLE_PREFIX}state_inst SET gmt_end = ?, excep = ?, status = ?, output_params = ?, gmt_updated = ? WHERE id = ? AND machine_inst_id = ?"
	UpdateStateExecutionStatusSql               = "UPDATE ${TABLE_PREFIX}state_inst SET status = ?, gmt_updated = ? WHERE machine_inst_id = ? AND id = ?"
	QueryStateInstancesByMachineInstanceIdSql   = "SELECT " + StateInstanceFields + " FROM ${TABLE_PREFIX}state_inst WHERE machine_inst_id = ? ORDER BY gmt_started, ID ASC"
	GetStateInstanceByIdAndMachineInstanceIdSql = "SELECT " + StateInstanceFields + " FROM ${TABLE_PREFIX}state_inst WHERE machine_inst_id = ? AND id = ?"
)

type StateLogStore struct {
	Store
	seqGenerator     sequence.SeqGenerator
	paramsSerializer serializer.ParamsSerializer
	errorSerializer  serializer.ErrorSerializer

	tablePrefix     string
	defaultTenantId string

	recordStateMachineStartedSql            string
	recordStateMachineFinishedSql           string
	updateStateMachineRunningStatusSql      string
	getStateMachineInstanceByIdSql          string
	getStateMachineInstanceByBusinessKeySql string
	queryStateMachineInstancesByParentIdSql string

	recordStateStartedSql                       string
	recordStateFinishedSql                      string
	updateStateExecutionStatusSql               string
	queryStateInstancesByMachineInstanceIdSql   string
	getStateInstanceByIdAndMachineInstanceIdSql string

	sagaTransactionalTemplate sagaTm.SagaTransactionalTemplate
}

func NewStateLogStore(db *sql.DB, tablePrefix string) *StateLogStore {
	r := regexp.MustCompile(TablePrefix)

	stateLogStore := &StateLogStore{
		Store: Store{db},

		seqGenerator:     sequence.NewUUIDSeqGenerator(),
		paramsSerializer: serializer.ParamsSerializer{},
		errorSerializer:  serializer.ErrorSerializer{},

		tablePrefix:     tablePrefix,
		defaultTenantId: "000001",

		recordStateMachineStartedSql:            r.ReplaceAllString(RecordStateMachineStartedSql, tablePrefix),
		recordStateMachineFinishedSql:           r.ReplaceAllString(RecordStateMachineFinishedSql, tablePrefix),
		updateStateMachineRunningStatusSql:      r.ReplaceAllString(UpdateStateMachineRunningStatusSql, tablePrefix),
		getStateMachineInstanceByIdSql:          r.ReplaceAllString(GetStateMachineInstanceByIdSql, tablePrefix),
		getStateMachineInstanceByBusinessKeySql: r.ReplaceAllString(GetStateMachineInstanceByBusinessKeySql, tablePrefix),
		queryStateMachineInstancesByParentIdSql: r.ReplaceAllString(QueryStateMachineInstancesByParentIdSql, tablePrefix),

		recordStateStartedSql:                       r.ReplaceAllString(RecordStateStartedSql, tablePrefix),
		recordStateFinishedSql:                      r.ReplaceAllString(RecordStateFinishedSql, tablePrefix),
		updateStateExecutionStatusSql:               r.ReplaceAllString(UpdateStateExecutionStatusSql, tablePrefix),
		queryStateInstancesByMachineInstanceIdSql:   r.ReplaceAllString(QueryStateInstancesByMachineInstanceIdSql, tablePrefix),
		getStateInstanceByIdAndMachineInstanceIdSql: r.ReplaceAllString(GetStateInstanceByIdAndMachineInstanceIdSql, tablePrefix),
	}

	return stateLogStore
}

func (s *StateLogStore) RecordStateMachineStarted(ctx context.Context, machineInstance statelang.StateMachineInstance,
	context process_ctrl.ProcessContext) error {
	if machineInstance == nil {
		return nil
	}

	var err error
	defer func() {
		if err != nil {
			log.Errorf("record state machine start error: %v, StateMachine %s, XID: %s",
				err, machineInstance.StateMachine().Name(), machineInstance.ID())
		}
	}()

	//if parentId is not null, machineInstance is a SubStateMachine, do not start a new global transaction,
	//use parent transaction instead.
	parentId := machineInstance.ParentID()

	if parentId == "" {
		// begin transaction
		err = s.beginTransaction(ctx, machineInstance, context)
		if err != nil {
			return err
		}
		if gtx, ok := context.GetVariable(constant.VarNameGlobalTx).(*tm.GlobalTransaction); ok && gtx != nil {
			log.Infof("SAGA GlobalBegin success, SM=%s, XID=%s", machineInstance.StateMachine().Name(), gtx.Xid)
		} else {
			log.Warnf("SAGA GlobalBegin missing GlobalTransaction in context for SM=%s", machineInstance.StateMachine().Name())
		}
	}

	if machineInstance.ID() == "" && s.seqGenerator != nil {
		machineInstance.SetID(s.seqGenerator.GenerateId(constant.SeqEntityStateMachineInst, ""))
	}

	// bind SAGA branch type
	context.SetVariable(constant2.BranchTypeKey, branch.BranchTypeSAGA)

	serializedStartParams, err := s.paramsSerializer.Serialize(machineInstance.StartParams())
	if err != nil {
		return err
	}
	machineInstance.SetSerializedStartParams(serializedStartParams)
	affected, err := ExecuteUpdate(s.db, s.recordStateMachineStartedSql, execStateMachineInstanceStatementForInsert, machineInstance)
	if err != nil {
		return err
	}
	if affected <= 0 {
		return errors.New("affected rows is smaller than 0")
	}

	log.Infof("RecordStateMachineStarted ok, SM=%s, XID=%s", machineInstance.StateMachine().Name(), machineInstance.ID())

	return nil
}

func (s *StateLogStore) beginTransaction(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error {
	if s.sagaTransactionalTemplate == nil {
		log.Debugf("begin transaction fail, sagaTransactionalTemplate is not existence")
		return nil
	}
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
	if !ok {
		return errors.New("begin transaction fail, stateMachineConfig is required in context")
	}

	defer func() {
		isAsync, ok := context.GetVariable(constant.VarNameIsAsyncExecution).(bool)
		if ok && isAsync {
			s.ClearUp(context)
		}
	}()

	txName := constant.SagaTransNamePrefix + machineInstance.StateMachine().Name()
	appId, group := rm.GetRmAppAndGroup()
	log.Infof("Begin SAGA global transaction: txName=%s, appId=%s, txServiceGroup=%s", txName, appId, group)
	gtx, err := s.sagaTransactionalTemplate.BeginTransaction(ctx, time.Duration(cfg.GetTransOperationTimeout()), txName)
	if err != nil {
		return err
	}
	xid := gtx.Xid
	machineInstance.SetID(xid)

	context.SetVariable(constant.VarNameGlobalTx, gtx)

	machineContext := machineInstance.Context()
	if machineContext != nil {
		machineContext[constant.VarNameGlobalTx] = gtx
	}
	log.Infof("Begin SAGA global transaction ok: XID=%s", xid)
	return nil
}

func (s *StateLogStore) RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance,
	context process_ctrl.ProcessContext) error {
	if machineInstance == nil {
		return nil
	}

	defer func() {
		s.ClearUp(context)
	}()

	endParams := machineInstance.EndParams()
	if endParams != nil {
		delete(endParams, constant.VarNameGlobalTx)
	}

	// If compensation ran, reconcile compensation status from DB state list to align with Java semantics
	if list, err := s.GetStateInstanceListByMachineInstanceId(machineInstance.ID()); err == nil && len(list) > 0 {
		compSeen := false
		compAllSU := true
		for _, si := range list {
			if si.StateIDCompensatedFor() != "" {
				compSeen = true
				if si.Status() != statelang.SU {
					compAllSU = false
					break
				}
			}
		}
		if compSeen {
			if compAllSU {
				machineInstance.SetCompensationStatus(statelang.SU)
				machineInstance.SetStatus(statelang.FA)
			} else if machineInstance.CompensationStatus() == "" || machineInstance.CompensationStatus() == statelang.RU {
				machineInstance.SetCompensationStatus(statelang.UN)
			}
		} else {
			// No compensation executed. If final status不是SU，则归一化为FA并清空补偿态。
			if machineInstance.Status() != statelang.SU {
				machineInstance.SetCompensationStatus("")
				machineInstance.SetStatus(statelang.FA)
			}
		}
	}

	// if success, clear exception
	if statelang.SU == machineInstance.Status() && machineInstance.Exception() != nil {
		machineInstance.SetException(nil)
	}

	serializedEndParams, err := s.paramsSerializer.Serialize(endParams)
	if err != nil {
		return err
	}
	machineInstance.SetSerializedEndParams(serializedEndParams)
	serializedError, err := s.errorSerializer.Serialize(machineInstance.Exception())
	if err != nil {
		return err
	}
	if len(serializedError) > 0 {
		machineInstance.SetSerializedError(serializedError)
	}

	affected, err := ExecuteUpdate(s.db, s.recordStateMachineFinishedSql, execStateMachineInstanceStatementForUpdate, machineInstance)
	if err != nil {
		return err
	}
	if affected <= 0 {
		// Reload and retry once with latest gmt_updated for optimistic lock
		current, _ := s.GetStateMachineInstance(machineInstance.ID())
		if current != nil {
			if !current.IsRunning() {
				log.Infof("StateMachineInstance[%s] already finished (no rows updated), skip duplicate finish", machineInstance.ID())
				return nil
			}
			// retry with refreshed updatedTime
			machineInstance.SetUpdatedTime(current.UpdatedTime())
			affected2, err2 := ExecuteUpdate(s.db, s.recordStateMachineFinishedSql, execStateMachineInstanceStatementForUpdate, machineInstance)
			if err2 != nil {
				return err2
			}
			if affected2 <= 0 {
				// Check again if it's already finished to avoid noisy warnings
				current2, _ := s.GetStateMachineInstance(machineInstance.ID())
				if current2 != nil && !current2.IsRunning() && !current2.EndTime().IsZero() {
					log.Infof("StateMachineInstance[%s] appears already finished after retry (no rows updated), treat as success", machineInstance.ID())
					return nil
				}
				// Fallback: update without gmt_updated predicate to guarantee terminal state is recorded
				// Use warn-level visibility as this is an abnormal path indicating potential concurrency issues
				log.Warnf("StateMachineInstance[%s] executing fallback finish (no optimistic lock) after retry failed, status=%s, comp=%s",
					machineInstance.ID(), machineInstance.Status(), machineInstance.CompensationStatus())
				// Build SQL by stripping the trailing ' and gmt_updated = ?'
				rawSql := s.recordStateMachineFinishedSql
				noWhereSql := strings.Replace(rawSql, " and gmt_updated = ?", "", 1)
				// Prepare parameters mirroring execStateMachineInstanceStatementForUpdate minus last arg
				var serializedError []byte
				if machineInstance.SerializedError() != nil && len(machineInstance.SerializedError().([]byte)) > 0 {
					serializedError = machineInstance.SerializedError().([]byte)
				}
				var compensationStatus sql.NullString
				if machineInstance.CompensationStatus() != "" {
					compensationStatus.Valid = true
					compensationStatus.String = string(machineInstance.CompensationStatus())
				}
				// end_time, excep, end_params, status, compensation_status, is_running, gmt_updated, id
				affectedFallback, errFallback := s.db.Exec(noWhereSql,
					machineInstance.EndTime(),
					serializedError,
					machineInstance.SerializedEndParams(),
					machineInstance.Status(),
					compensationStatus,
					machineInstance.IsRunning(),
					time.Now(),
					machineInstance.ID(),
				)
				if errFallback != nil {
					log.Errorf("StateMachineInstance[%s] fallback finish failed: %v, status=%s, comp=%s",
						machineInstance.ID(), errFallback,
						machineInstance.Status(), machineInstance.CompensationStatus())
					return errFallback
				}
				rowsAffected, _ := affectedFallback.RowsAffected()
				if rowsAffected <= 0 {
					log.Warnf("StateMachineInstance[%s] fallback finish affected 0 rows, may indicate concurrent update or missing record",
						machineInstance.ID())
				}
				// reflect latest update time locally to avoid timeout false positives
				machineInstance.SetUpdatedTime(time.Now())
			}
		} else {
			log.Debugf("StateMachineInstance[%s] finish update affected 0 rows and reload failed; skipping retry", machineInstance.ID())
		}
	} else {
		// reflect latest update time locally to avoid timeout false positives
		machineInstance.SetUpdatedTime(time.Now())
	}

	// check if timeout or else report transaction finished
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
	if !ok {
		return errors.New("stateMachineConfig is required in context")
	}

	// UpdatedTime recently refreshed at successful finish; this check should only catch real timeouts
	if pcext.IsTimeout(machineInstance.UpdatedTime(), cfg.GetTransOperationTimeout()) {
		log.Warnf("StateMachineInstance[%s] is execution timeout, skip report transaction finished to server.", machineInstance.ID())
	} else if machineInstance.ParentID() == "" {
		//if parentId is not null, machineInstance is a SubStateMachine, do not report global transaction.
		err = s.reportTransactionFinished(ctx, machineInstance, context)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *StateLogStore) reportTransactionFinished(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error {
	defer func() {
		s.ClearUp(context)
	}()

	if s.sagaTransactionalTemplate == nil {
		log.Debugf("report transaction finished fail, sagaTransactionalTemplate is not existence")
		return nil
	}
	globalTransaction, err := s.getGlobalTransaction(ctx, machineInstance, context)
	if err != nil {
		log.Errorf("Failed to get global transaction: %v, StateMachine: %s, XID: %s",
			err, machineInstance.StateMachine().Name(), machineInstance.ID())
		// Align with Java semantics: getGlobalTransaction failure is non-fatal to state machine finish
		return nil
	}

	var globalStatus message.GlobalStatus
	if statelang.SU == machineInstance.Status() && machineInstance.CompensationStatus() == "" {
		globalStatus = message.GlobalStatusCommitted
	} else if statelang.SU == machineInstance.CompensationStatus() {
		globalStatus = message.GlobalStatusRollbacked
	} else if statelang.FA == machineInstance.CompensationStatus() || statelang.UN == machineInstance.CompensationStatus() {
		globalStatus = message.GlobalStatusRollbackRetrying
	} else if statelang.FA == machineInstance.Status() && machineInstance.CompensationStatus() == "" {
		globalStatus = message.GlobalStatusFinished
	} else if statelang.UN == machineInstance.Status() && machineInstance.CompensationStatus() == "" {
		globalStatus = message.GlobalStatusCommitRetrying
	} else {
		globalStatus = message.GlobalStatusUnKnown
	}

	globalTransaction.TxStatus = globalStatus
	err = s.sagaTransactionalTemplate.ReportTransaction(ctx, globalTransaction)
	if err != nil {
		// Enhanced error logging aligned with Java implementation (DbAndReportTcStateLogStore.java:246-261)
		log.Errorf("Report transaction finish to server failed: StateMachine=%s, XID=%s, Status=%v, Err=%v",
			machineInstance.StateMachine().Name(),
			machineInstance.ID(),
			globalStatus,
			err)
		// Align with Java semantics: reporting to TC should not fail the local state machine finish.
		// Swallow error to keep success path green.
		return nil
	}
	return nil
}

func (s *StateLogStore) getGlobalTransaction(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) (*tm.GlobalTransaction, error) {
	globalTransaction, ok := context.GetVariable(constant.VarNameGlobalTx).(*tm.GlobalTransaction)
	if ok {
		return globalTransaction, nil
	}

	var xid string
	parentId := machineInstance.ParentID()
	if parentId == "" {
		xid = machineInstance.ID()
	} else {
		xid = parentId[:strings.LastIndex(parentId, constant.SeperatorParentId)]
	}
	globalTransaction, err := s.sagaTransactionalTemplate.ReloadTransaction(ctx, xid)
	if err != nil {
		return nil, err
	}
	context.SetVariable(constant.VarNameGlobalTx, globalTransaction)
	return globalTransaction, nil
}

func (s *StateLogStore) RecordStateMachineRestarted(ctx context.Context, machineInstance statelang.StateMachineInstance,
	context process_ctrl.ProcessContext) error {
	if machineInstance == nil {
		return nil
	}
	updated := time.Now()
	affected, err := ExecuteUpdateArgs(s.db, s.updateStateMachineRunningStatusSql,
		machineInstance.IsRunning(), updated, machineInstance.ID(), machineInstance.UpdatedTime())
	if err != nil {
		return err
	}
	if affected <= 0 {
		return errors.New(fmt.Sprintf("StateMachineInstance [id:%s] is recovered by another execution, restart denied",
			machineInstance.ID()))
	}
	machineInstance.SetUpdatedTime(updated)
	return nil
}

func (s *StateLogStore) RecordStateStarted(ctx context.Context, stateInstance statelang.StateInstance,
	context process_ctrl.ProcessContext) error {
	if stateInstance == nil {
		return nil
	}
	// ensure compensation linkage is restored when upstream context loses the original state id
	if stateInstance.StateIDCompensatedFor() == "" {
		if holder, ok := context.GetVariable(constant.VarNameCurrentCompensationHolder).(*pcext.CompensationHolder); ok && holder != nil {
			if toCompensate, okLoad := holder.StatesNeedCompensation().Load(stateInstance.Name()); okLoad {
				if original, okInst := toCompensate.(statelang.StateInstance); okInst && original != nil {
					if original.ID() != "" {
						stateInstance.SetStateIDCompensatedFor(original.ID())
						stateInstance.SetCompensationState(original)
						holder.StatesForCompensation().Store(stateInstance.Name(), stateInstance)
					}
				}
			}
		}
	}
	isUpdateMode, err := s.isUpdateMode(stateInstance, context)
	if err != nil {
		return err
	}

	// if this state is for retry, do not register branch
	if stateInstance.StateIDRetriedFor() != "" {
		if isUpdateMode {
			stateInstance.SetID(stateInstance.StateIDRetriedFor())
		} else {
			// generate id by default
			stateInstance.SetID(s.generateRetryStateInstanceId(stateInstance))
		}
	} else if stateInstance.StateIDCompensatedFor() != "" {
		// if this state is for compensation, do not register branch
		stateInstance.SetID(s.generateCompensateStateInstanceId(stateInstance, isUpdateMode))
	} else {
		// register branch
		sm := stateInstance.StateMachineInstance().StateMachine().Name()
		log.Infof("Register SAGA branch begin, SM=%s, state=%s", sm, stateInstance.Name())
		if err := s.branchRegister(ctx, stateInstance, context); err != nil {
			log.Errorf("Register SAGA branch failed, SM=%s, state=%s, err=%v", sm, stateInstance.Name(), err)
			return err
		}
		log.Infof("Register SAGA branch ok, SM=%s, state=%s, branchId=%s", sm, stateInstance.Name(), stateInstance.ID())
	}

	if stateInstance.ID() == "" && s.seqGenerator != nil {
		stateInstance.SetID(s.seqGenerator.GenerateId(constant.SeqEntityStateInst, ""))
	}

	serializedParams, err := s.paramsSerializer.Serialize(stateInstance.InputParams())
	if err != nil {
		return err
	}
	stateInstance.SetSerializedInputParams(serializedParams)

	var affected int64
	if !isUpdateMode {
		affected, err = ExecuteUpdate(s.db, s.recordStateStartedSql, execStateInstanceStatementForInsert, stateInstance)
	} else {
		affected, err = ExecuteUpdateArgs(s.db, s.updateStateExecutionStatusSql,
			stateInstance.Status(), time.Now(), stateInstance.StateMachineInstance().ID(), stateInstance.ID())
	}

	if err != nil {
		return err
	}
	if affected <= 0 {
		return errors.New("affected rows is smaller than 0")
	}
	log.Infof("RecordStateStarted ok, SM=%s, state=%s, branchId=%s", stateInstance.StateMachineInstance().StateMachine().Name(), stateInstance.Name(), stateInstance.ID())
	return nil
}

func (s *StateLogStore) isUpdateMode(stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) (bool, error) {
	instruction, ok := context.GetInstruction().(*pcext.StateInstruction)
	if !ok {
		return false, errors.New("stateInstruction is required in processContext")
	}
	instructionState, err := instruction.GetState(context)
	if err != nil {
		return false, err
	}
	taskState, _ := instructionState.(*state.AbstractTaskState)
	stateMachine := stateInstance.StateMachineInstance().StateMachine()

	if stateInstance.StateIDRetriedFor() != "" {
		if taskState != nil && taskState.RetryPersistModeUpdate() {
			return true, nil
		}
		if stateMachine != nil && stateMachine.IsRetryPersistModeUpdate() {
			return true, nil
		}
		return false, nil
	} else if stateInstance.StateIDCompensatedFor() != "" {
		// find if this compensate has been executed
		stateList := stateInstance.StateMachineInstance().StateList()
		for _, instance := range stateList {
			if instance.IsForCompensation() && instance.Name() == stateInstance.Name() {
				if taskState != nil && taskState.CompensatePersistModeUpdate() {
					return true, nil
				}
				if stateMachine != nil && stateMachine.IsCompensatePersistModeUpdate() {
					return true, nil
				}
				return false, nil
			}
		}
	}
	return false, nil
}

func (s *StateLogStore) generateRetryStateInstanceId(stateInstance statelang.StateInstance) string {
	originalStateInstId := stateInstance.StateIDRetriedFor()
	maxIndex := 1
	machineInstance := stateInstance.StateMachineInstance()
	originalStateInst := machineInstance.State(originalStateInstId)
	for originalStateInst.StateIDRetriedFor() != "" {
		originalStateInst = machineInstance.State(originalStateInst.StateIDRetriedFor())
		idIndex := s.getIdIndex(originalStateInst.ID(), ".")
		if idIndex > maxIndex {
			maxIndex = idIndex
		}
		originalStateInstId = originalStateInst.ID()
	}
	return fmt.Sprintf("%s.%d", originalStateInstId, maxIndex)
}

func (s *StateLogStore) generateCompensateStateInstanceId(stateInstance statelang.StateInstance, isUpdateMode bool) string {
	originalCompensateStateInstId := stateInstance.StateIDCompensatedFor()
	maxIndex := 1
	if isUpdateMode {
		return fmt.Sprintf("%s-%d", originalCompensateStateInstId, maxIndex)
	}

	machineInstance := stateInstance.StateMachineInstance()
	for _, aStateInstance := range machineInstance.StateList() {
		if aStateInstance != stateInstance && originalCompensateStateInstId == aStateInstance.StateIDCompensatedFor() {
			idIndex := s.getIdIndex(aStateInstance.ID(), "-")
			if idIndex > maxIndex {
				maxIndex = idIndex
			}
			maxIndex++
		}
	}
	return fmt.Sprintf("%s-%d", originalCompensateStateInstId, maxIndex)
}

func safeGetSagaRM() (rm.ResourceManager, bool) {
	defer func() { recover() }()
	return rm.GetRmCacheInstance().GetResourceManager(branch.BranchTypeSAGA), true
}

func (s *StateLogStore) branchRegister(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error {

	//Register branch
	var err error
	machineInstance := stateInstance.StateMachineInstance()
	defer func() {
		if err != nil {
			log.Errorf("Branch transaction failure. StateMachine: %s, XID: %s, State: %s, stateId: %s, err: %v",
				machineInstance.StateMachine().Name(), machineInstance.ID(), stateInstance.Name(), stateInstance.ID(), err)
		}
	}()

	if cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig); ok {
		if !cfg.IsSagaBranchRegisterEnable() {
			if stateInstance.ID() == "" {
				if seq := cfg.SeqGenerator(); seq != nil {
					stateInstance.SetID(seq.GenerateId(constant.SeqEntityStateInst, ""))
				}
				if stateInstance.ID() == "" {
					stateInstance.SetID(fmt.Sprintf("%s-%d", stateInstance.Name(), time.Now().UnixNano()))
				}
			}
			return nil
		}
	}

	globalTransaction, err := s.getGlobalTransaction(ctx, machineInstance, context)
	if err != nil {
		if _, ok := engExc.IsEngineExecutionException(err); ok {
			return err
		}
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchRegisterFailed,
			fmt.Sprintf("get global transaction failed, stateMachine=%s, state=%s",
				machineInstance.StateMachine().Name(), stateInstance.Name()), err)
		return err
	}
	if globalTransaction == nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeGlobalTransactionNotExist,
			fmt.Sprintf("global transaction does not exist, stateMachine=%s, state=%s",
				machineInstance.StateMachine().Name(), stateInstance.Name()), nil)
		return err
	}

	// For SAGA, resourceId should be applicationId#txServiceGroup (aligned with SagaResource)
	appId, group := rm.GetRmAppAndGroup()
	resourceId := appId + "#" + group
	// Prefer RM ResourceManager (BranchTypeSAGA) for branch register
	if mgr, ok := safeGetSagaRM(); ok && mgr != nil {
		bid, e := mgr.BranchRegister(ctx, rm.BranchRegisterParam{
			BranchType:      branch.BranchTypeSAGA,
			ResourceId:      resourceId,
			Xid:             globalTransaction.Xid,
			ClientId:        "",
			ApplicationData: "",
			LockKeys:        "",
		})
		if e != nil {
			err = engExc.NewEngineExecutionException(
				seataErrors.TransactionErrorCodeBranchRegisterFailed,
				fmt.Sprintf("branch register via rm failed, stateMachine=%s, state=%s, xid=%s",
					machineInstance.StateMachine().Name(), stateInstance.Name(), globalTransaction.Xid), e)
			return err
		}
		if bid <= 0 {
			err = engExc.NewEngineExecutionException(
				seataErrors.TransactionErrorCodeBranchRegisterFailed,
				fmt.Sprintf("branch register returned invalid branchId (<=0), stateMachine=%s, state=%s, xid=%s",
					machineInstance.StateMachine().Name(), stateInstance.Name(), globalTransaction.Xid), nil)
			return err
		}
		stateInstance.SetID(strconv.FormatInt(bid, 10))
		return nil
	}
	if s.sagaTransactionalTemplate == nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchRegisterFailed,
			"saga transactional template is not initialized", nil)
		return err
	}
	branchId, err := s.sagaTransactionalTemplate.BranchRegister(ctx, resourceId, "", globalTransaction.Xid, "", "")
	if err != nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchRegisterFailed,
			fmt.Sprintf("branch register via template failed, stateMachine=%s, state=%s, xid=%s",
				machineInstance.StateMachine().Name(), stateInstance.Name(), globalTransaction.Xid), err)
		return err
	}
	if branchId <= 0 {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchRegisterFailed,
			fmt.Sprintf("branch register returned invalid branchId (<=0), stateMachine=%s, state=%s, xid=%s",
				machineInstance.StateMachine().Name(), stateInstance.Name(), globalTransaction.Xid), nil)
		return err
	}

	stateInstance.SetID(strconv.FormatInt(branchId, 10))
	return nil
}

func (s *StateLogStore) getIdIndex(stateInstanceId string, separator string) int {
	if stateInstanceId != "" {
		start := strings.LastIndex(stateInstanceId, separator)
		if start > 0 {
			indexStr := stateInstanceId[start+1:]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				log.Warnf("get stateInstance id index failed %v", err)
				return -1
			}
			return index
		}
	}
	return -1
}

func (s *StateLogStore) RecordStateFinished(ctx context.Context, stateInstance statelang.StateInstance,
	context process_ctrl.ProcessContext) error {
	if stateInstance == nil {
		return nil
	}

	serializedOutputParams, err := s.paramsSerializer.Serialize(stateInstance.OutputParams())
	if err != nil {
		return err
	}
	stateInstance.SetSerializedOutputParams(serializedOutputParams)

	serializedError, err := s.errorSerializer.Serialize(stateInstance.Error())
	if err != nil {
		return err
	}
	stateInstance.SetSerializedError(serializedError)

	affected, err := ExecuteUpdate(s.db, s.recordStateFinishedSql, execStateInstanceStatementForUpdate, stateInstance)
	if err != nil {
		return err
	}
	if affected <= 0 {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeFailedWriteSession,
			fmt.Sprintf("state finish update affected 0 rows, possible concurrent update lost. state=%s id=%s machine=%s",
				stateInstance.Name(), stateInstance.ID(), stateInstance.StateMachineInstance().ID()), nil)
		return err
	}

	if cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig); ok {
		if !cfg.IsRmReportSuccessEnable() && stateInstance.Status() == statelang.SU && stateInstance.CompensationStatus() == "" {
			return nil
		}
	}

	// always report branch on state finish when enabled
	return s.branchReport(ctx, stateInstance, context)

}

func (s *StateLogStore) branchReport(ctx context.Context, stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error {

	var branchStatus branch.BranchStatus
	// find out the original state instance, only the original state instance is registered on the server,
	// and its status should be reported.
	var originalStateInst statelang.StateInstance
	if stateInstance.StateIDRetriedFor() != "" {
		isUpdateMode, err := s.isUpdateMode(stateInstance, context)
		if err != nil {
			return err
		}

		if isUpdateMode {
			originalStateInst = stateInstance
		} else {
			originalStateInst = s.findOutOriginalStateInstanceOfRetryState(stateInstance)
		}

		if statelang.SU == stateInstance.Status() {
			branchStatus = branch.BranchStatusPhasetwoCommitted
		} else if statelang.FA == stateInstance.Status() || statelang.UN == stateInstance.Status() {
			branchStatus = branch.BranchStatusPhaseoneFailed
		} else {
			branchStatus = branch.BranchStatusUnknown
		}
	} else if stateInstance.StateIDCompensatedFor() != "" {
		isUpdateMode, err := s.isUpdateMode(stateInstance, context)
		if err != nil {
			return err
		}

		if isUpdateMode {
			originalStateInst = stateInstance.StateMachineInstance().StateMap()[stateInstance.StateIDCompensatedFor()]
		} else {
			originalStateInst = s.findOutOriginalStateInstanceOfCompensateState(stateInstance)
		}
	}

	if originalStateInst == nil {
		originalStateInst = stateInstance
	}

	if branchStatus == branch.BranchStatusUnknown {
		if statelang.SU == originalStateInst.Status() && originalStateInst.CompensationStatus() == "" {
			branchStatus = branch.BranchStatusPhasetwoCommitted
		} else if statelang.SU == originalStateInst.CompensationStatus() {
			branchStatus = branch.BranchStatusPhasetwoRollbacked
		} else if statelang.FA == originalStateInst.CompensationStatus() || statelang.UN == originalStateInst.CompensationStatus() {
			branchStatus = branch.BranchStatusPhasetwoRollbackFailedRetryable
		} else if (statelang.FA == originalStateInst.Status() || statelang.UN == originalStateInst.Status()) && originalStateInst.CompensationStatus() == "" {
			branchStatus = branch.BranchStatusPhaseoneFailed
		} else {
			branchStatus = branch.BranchStatusUnknown
		}
	}

	var err error
	defer func() {
		if err != nil {
			log.Errorf("Report branch status to server error:%s, StateMachine:%s, StateName:%s, XID:%s, branchId:%s, branchStatus:%s, err:%v",
				err.Error(), originalStateInst.StateMachineInstance().StateMachine().Name(), originalStateInst.Name(),
				originalStateInst.StateMachineInstance().ID(), originalStateInst.ID(), branchStatus, err)
		}
	}()

	globalTransaction, err := s.getGlobalTransaction(ctx, stateInstance.StateMachineInstance(), context)
	if err != nil {
		if _, ok := engExc.IsEngineExecutionException(err); ok {
			return err
		}
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchReportFailed,
			fmt.Sprintf("get global transaction failed for branch report, stateMachine=%s, state=%s",
				stateInstance.StateMachineInstance().StateMachine().Name(), stateInstance.Name()), err)
		return err
	}
	if globalTransaction == nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeGlobalTransactionNotExist,
			fmt.Sprintf("global transaction does not exist for branch report, stateMachine=%s, state=%s",
				stateInstance.StateMachineInstance().StateMachine().Name(), stateInstance.Name()), nil)
		return err
	}

	log.Infof("BranchReport prepare, XID=%s, branchId(raw)=%s, status=%s", globalTransaction.Xid, originalStateInst.ID(), branchStatus)
	branchId, perr := strconv.ParseInt(originalStateInst.ID(), 10, 64)
	if perr != nil {
		return engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchReportFailed,
			fmt.Sprintf("invalid branchId '%s' (must be numeric); ensure GlobalBegin + BranchRegister succeeded and resourceId is 'applicationId#txServiceGroup'", originalStateInst.ID()),
			perr,
		)
	}
	if mgr := rm.GetRmCacheInstance().GetResourceManager(branch.BranchTypeSAGA); mgr != nil {
		log.Infof("BranchReport via SagaResourceManager, XID=%s, branchId=%d, status=%s", globalTransaction.Xid, branchId, branchStatus)
		if err = mgr.BranchReport(ctx, rm.BranchReportParam{
			BranchType:      branch.BranchTypeSAGA,
			Xid:             globalTransaction.Xid,
			BranchId:        branchId,
			Status:          branchStatus,
			ApplicationData: "",
		}); err != nil {
			err = engExc.NewEngineExecutionException(
				seataErrors.TransactionErrorCodeBranchReportFailed,
				fmt.Sprintf("branch report via rm failed, stateMachine=%s, state=%s, xid=%s, branchId=%d",
					originalStateInst.StateMachineInstance().StateMachine().Name(), originalStateInst.Name(), globalTransaction.Xid, branchId),
				err,
			)
			return err
		}
		return nil
	}
	if s.sagaTransactionalTemplate == nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchReportFailed,
			"saga transactional template is not initialized", nil)
		return err
	}
	log.Infof("BranchReport via SagaTransactionalTemplate, XID=%s, branchId=%d, status=%s", globalTransaction.Xid, branchId, branchStatus)
	if err = s.sagaTransactionalTemplate.BranchReport(ctx, globalTransaction.Xid, branchId, branchStatus, ""); err != nil {
		err = engExc.NewEngineExecutionException(
			seataErrors.TransactionErrorCodeBranchReportFailed,
			fmt.Sprintf("branch report via template failed, stateMachine=%s, state=%s, xid=%s, branchId=%d",
				originalStateInst.StateMachineInstance().StateMachine().Name(), originalStateInst.Name(), globalTransaction.Xid, branchId),
			err,
		)
		return err
	}
	return nil
}

func (s *StateLogStore) findOutOriginalStateInstanceOfRetryState(stateInstance statelang.StateInstance) statelang.StateInstance {
	stateMap := stateInstance.StateMachineInstance().StateMap()
	originalStateInst := stateMap[stateInstance.StateIDRetriedFor()]
	for originalStateInst.StateIDRetriedFor() != "" {
		originalStateInst = stateMap[stateInstance.StateIDRetriedFor()]
	}
	return originalStateInst
}

func (s *StateLogStore) findOutOriginalStateInstanceOfCompensateState(stateInstance statelang.StateInstance) statelang.StateInstance {
	stateMap := stateInstance.StateMachineInstance().StateMap()
	originalStateInst := stateMap[stateInstance.StateIDCompensatedFor()]
	for originalStateInst.StateIDRetriedFor() != "" {
		originalStateInst = stateMap[stateInstance.StateIDRetriedFor()]
	}
	return originalStateInst
}

func (s *StateLogStore) GetStateMachineInstance(stateMachineInstanceId string) (statelang.StateMachineInstance, error) {
	stateMachineInstance, err := SelectOne(s.db, s.getStateMachineInstanceByIdSql, scanRowsToStateMachineInstance,
		stateMachineInstanceId)
	if err != nil {
		return stateMachineInstance, err
	}
	if stateMachineInstance == nil {
		return nil, nil
	}

	stateInstanceList, err := s.GetStateInstanceListByMachineInstanceId(stateMachineInstanceId)
	if err != nil {
		return stateMachineInstance, err
	}
	for _, stateInstance := range stateInstanceList {
		stateMachineInstance.PutState(stateInstance.ID(), stateInstance)
	}
	err = s.deserializeStateMachineParamsAndException(stateMachineInstance)
	return stateMachineInstance, err
}

func (s *StateLogStore) GetStateMachineInstanceByBusinessKey(businessKey string, tenantId string) (statelang.StateMachineInstance, error) {
	if tenantId == "" {
		tenantId = s.defaultTenantId
	}
	stateMachineInstance, err := SelectOne(s.db, s.getStateMachineInstanceByBusinessKeySql,
		scanRowsToStateMachineInstance, businessKey, tenantId)
	if err != nil || stateMachineInstance == nil {
		return stateMachineInstance, err
	}
	stateInstanceList, err := s.GetStateInstanceListByMachineInstanceId(stateMachineInstance.ID())
	if err != nil {
		return stateMachineInstance, err
	}
	for _, stateInstance := range stateInstanceList {
		stateMachineInstance.PutState(stateInstance.ID(), stateInstance)
	}
	err = s.deserializeStateMachineParamsAndException(stateMachineInstance)
	return stateMachineInstance, err
}

func (s *StateLogStore) deserializeStateMachineParamsAndException(stateMachineInstance statelang.StateMachineInstance) error {
	if stateMachineInstance == nil {
		return nil
	}

	serializedError := stateMachineInstance.SerializedError().([]byte)
	if serializedError != nil && len(serializedError) > 0 {
		deserializedError, err := s.errorSerializer.Deserialize(serializedError)
		if err != nil {
			return err
		}
		stateMachineInstance.SetException(deserializedError)
	}

	serializedStartParams := stateMachineInstance.SerializedStartParams()
	if serializedStartParams != nil && serializedStartParams != "" {
		startParams, err := s.paramsSerializer.Deserialize(serializedStartParams.(string))
		if err != nil {
			return err
		}
		stateMachineInstance.SetStartParams(startParams.(map[string]any))
	}

	serializedOutputParams := stateMachineInstance.SerializedEndParams()
	if serializedOutputParams != nil && serializedOutputParams != "" {
		endParams, err := s.paramsSerializer.Deserialize(serializedOutputParams.(string))
		if err != nil {
			return err
		}
		stateMachineInstance.SetEndParams(endParams.(map[string]any))
	}
	return nil
}

func (s *StateLogStore) GetStateMachineInstanceByParentId(parentId string) ([]statelang.StateMachineInstance, error) {
	return SelectList(s.db, s.queryStateMachineInstancesByParentIdSql, scanRowsToStateMachineInstance, parentId)
}

func (s *StateLogStore) GetStateInstance(stateInstanceId string, stateMachineInstanceId string) (statelang.StateInstance, error) {
	stateInstance, err := SelectOne(s.db, s.getStateInstanceByIdAndMachineInstanceIdSql, scanRowsToStateInstance,
		stateMachineInstanceId, stateInstanceId)
	if err != nil {
		return stateInstance, err
	}
	err = s.deserializeStateParamsAndException(stateInstance)
	return stateInstance, err
}

func (s *StateLogStore) GetStateInstanceListByMachineInstanceId(stateMachineInstanceId string) ([]statelang.StateInstance, error) {
	stateInstanceList, err := SelectList(s.db, s.queryStateInstancesByMachineInstanceIdSql,
		scanRowsToStateInstance, stateMachineInstanceId)
	if err != nil || len(stateInstanceList) == 0 {
		return stateInstanceList, err
	}

	lastStateInstance := stateInstanceList[len(stateInstanceList)-1]
	if lastStateInstance.EndTime().IsZero() {
		lastStateInstance.SetStatus(statelang.RU)
	}

	originStateMap := make(map[string]statelang.StateInstance)
	compensatedStateMap := make(map[string]statelang.StateInstance)
	retriedStateMap := make(map[string]statelang.StateInstance)

	for _, tempStateInstance := range stateInstanceList {
		err := s.deserializeStateParamsAndException(tempStateInstance)
		if err != nil {
			return stateInstanceList, err
		}

		if tempStateInstance.StateIDCompensatedFor() != "" {
			s.putLastStateToMap(compensatedStateMap, tempStateInstance, tempStateInstance.StateIDCompensatedFor())
		} else {
			if tempStateInstance.StateIDRetriedFor() != "" {
				s.putLastStateToMap(retriedStateMap, tempStateInstance, tempStateInstance.StateIDRetriedFor())
			}
			originStateMap[tempStateInstance.ID()] = tempStateInstance
		}
	}

	if len(compensatedStateMap) != 0 {
		for _, originState := range originStateMap {
			originState.SetCompensationState(compensatedStateMap[originState.ID()])
		}
	}

	if len(retriedStateMap) != 0 {
		for _, originState := range originStateMap {
			if _, ok := retriedStateMap[originState.ID()]; ok {
				originState.SetIgnoreStatus(true)
			}
		}
	}

	return stateInstanceList, nil
}

func (s *StateLogStore) putLastStateToMap(resultMap map[string]statelang.StateInstance, newState statelang.StateInstance, key string) {
	existed, ok := resultMap[key]
	if !ok {
		resultMap[key] = newState
	} else if newState.EndTime().After(existed.EndTime()) {
		existed.SetIgnoreStatus(true)
		resultMap[key] = newState
	} else {
		newState.SetIgnoreStatus(true)
	}
}

func (s *StateLogStore) deserializeStateParamsAndException(stateInstance statelang.StateInstance) error {
	if stateInstance == nil {
		return nil
	}
	serializedInputParams := stateInstance.SerializedInputParams()
	if serializedInputParams != nil && serializedInputParams != "" {
		inputParams, err := s.paramsSerializer.Deserialize(serializedInputParams.(string))
		if err != nil {
			return err
		}
		stateInstance.SetInputParams(inputParams)
	}
	serializedOutputParams := stateInstance.SerializedOutputParams()
	if serializedOutputParams != nil && serializedOutputParams != "" {
		outputParams, err := s.paramsSerializer.Deserialize(serializedOutputParams.(string))
		if err != nil {
			return err
		}
		stateInstance.SetOutputParams(outputParams)
	}
	serializedError := stateInstance.SerializedError().([]byte)
	if serializedError != nil {
		deserializedError, err := s.errorSerializer.Deserialize(serializedError)
		if err != nil {
			return err
		}
		stateInstance.SetError(deserializedError)
	}
	return nil
}

func (s *StateLogStore) SetSeqGenerator(seqGenerator sequence.SeqGenerator) {
	s.seqGenerator = seqGenerator
}

func (s *StateLogStore) ClearUp(context process_ctrl.ProcessContext) {
	context.RemoveVariable(constant2.XidKey)
	context.RemoveVariable(constant2.BranchTypeKey)
}

func (s *StateLogStore) SetSagaTransactionalTemplate(sagaTransactionalTemplate sagaTm.SagaTransactionalTemplate) {
	s.sagaTransactionalTemplate = sagaTransactionalTemplate
}
func (s *StateLogStore) GetSagaTransactionalTemplate() sagaTm.SagaTransactionalTemplate {
	return s.sagaTransactionalTemplate
}

func execStateMachineInstanceStatementForInsert(obj statelang.StateMachineInstance, stmt *sql.Stmt) (int64, error) {
	result, err := stmt.Exec(
		obj.ID(),
		obj.MachineID(),
		obj.TenantID(),
		obj.ParentID(),
		obj.StartedTime(),
		obj.BusinessKey(),
		obj.SerializedStartParams(),
		obj.IsRunning(),
		obj.Status(),
		obj.UpdatedTime(),
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func execStateMachineInstanceStatementForUpdate(obj statelang.StateMachineInstance, stmt *sql.Stmt) (int64, error) {
	var serializedError []byte
	if obj.SerializedError() != nil && len(obj.SerializedError().([]byte)) > 0 {
		serializedError = obj.SerializedError().([]byte)
	}
	var compensationStatus sql.NullString
	if obj.CompensationStatus() != "" {
		compensationStatus.Valid = true
		compensationStatus.String = string(obj.CompensationStatus())
	}

	result, err := stmt.Exec(
		obj.EndTime(),
		serializedError,
		obj.SerializedEndParams(),
		obj.Status(),
		compensationStatus,
		obj.IsRunning(),
		time.Now(),
		obj.ID(),
		obj.UpdatedTime(),
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func execStateInstanceStatementForInsert(obj statelang.StateInstance, stmt *sql.Stmt) (int64, error) {
	result, err := stmt.Exec(
		obj.ID(),
		obj.StateMachineInstance().ID(),
		obj.Name(),
		obj.Type(),
		obj.StartedTime(),
		obj.ServiceName(),
		obj.ServiceMethod(),
		obj.ServiceType(),
		obj.IsForUpdate(),
		obj.SerializedInputParams(),
		obj.Status(),
		obj.BusinessKey(),
		obj.StateIDCompensatedFor(),
		obj.StateIDRetriedFor(),
		obj.UpdatedTime(),
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func execStateInstanceStatementForUpdate(obj statelang.StateInstance, stmt *sql.Stmt) (int64, error) {
	var serializedError []byte
	if obj.SerializedError() != nil && len(obj.SerializedError().([]byte)) > 0 {
		serializedError = obj.SerializedError().([]byte)
	}

	updatedTime := obj.UpdatedTime()
	if updatedTime.IsZero() {
		updatedTime = time.Now()
	}
	machineInstanceID := obj.MachineInstanceID()
	if machineInstanceID == "" {
		if sm := obj.StateMachineInstance(); sm != nil {
			machineInstanceID = sm.ID()
		}
	}
	result, err := stmt.Exec(
		obj.EndTime(),
		serializedError,
		obj.Status(),
		obj.SerializedOutputParams(),
		updatedTime,
		obj.ID(),
		machineInstanceID,
	)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := result.RowsAffected()
	return rowsAffected, err
}

func scanRowsToStateMachineInstance(rows *sql.Rows) (statelang.StateMachineInstance, error) {
	stateMachineInstance := statelang.NewStateMachineInstanceImpl()
	var id, machineId, tenantId, parentId, businessKey, started, end, status, compensationStatus,
		updated, startParams, endParams sql.NullString
	var isRunning sql.NullBool
	var errorBlob []byte
	columns, _ := rows.Columns()

	args := []any{&id, &machineId, &tenantId, &parentId, &businessKey, &started, &end, &status,
		&compensationStatus, &isRunning, &updated}
	if len(columns) > 11 {
		args = append(args, &startParams, &endParams, &errorBlob)
	}

	err := rows.Scan(args...)
	if err != nil {
		return nil, err
	}

	if id.Valid {
		stateMachineInstance.SetID(id.String)
	}
	if machineId.Valid {
		stateMachineInstance.SetMachineID(machineId.String)
	}
	if tenantId.Valid {
		stateMachineInstance.SetTenantID(tenantId.String)
	}
	if parentId.Valid {
		stateMachineInstance.SetParentID(parentId.String)
	}
	if businessKey.Valid {
		stateMachineInstance.SetBusinessKey(businessKey.String)
	}
	if started.Valid {
		startedTime, _ := time.Parse(TimeLayout, started.String)
		stateMachineInstance.SetStartedTime(startedTime)
	}
	if end.Valid {
		endTime, _ := time.Parse(TimeLayout, end.String)
		stateMachineInstance.SetEndTime(endTime)
	}
	if status.Valid {
		stateMachineInstance.SetStatus(statelang.ExecutionStatus(status.String))
	}

	if compensationStatus.Valid {
		if compensationStatus.String != "" {
			stateMachineInstance.SetCompensationStatus(statelang.ExecutionStatus(compensationStatus.String))
		}
	}
	if isRunning.Valid {
		stateMachineInstance.SetRunning(isRunning.Bool)
	}
	if updated.Valid {
		updatedTime, _ := time.Parse(TimeLayout, updated.String)
		stateMachineInstance.SetUpdatedTime(updatedTime)
	}

	if len(columns) > 11 {
		if startParams.Valid {
			stateMachineInstance.SetSerializedStartParams(startParams.String)
		}
		if endParams.Valid {
			stateMachineInstance.SetSerializedEndParams(endParams.String)
		}
		stateMachineInstance.SetSerializedError(errorBlob)
	}
	return stateMachineInstance, nil
}

func scanRowsToStateInstance(rows *sql.Rows) (statelang.StateInstance, error) {
	stateInstance := statelang.NewStateInstanceImpl()
	var id, machineInstId, name, t, businessKey, started, serviceName, serviceMethod, serviceType, status,
		inputParams, outputParams, end, stateIdCompensatedFor, stateIdRetriedFor sql.NullString
	var isForUpdate sql.NullBool
	var errorBlob []byte
	err := rows.Scan(&id, &machineInstId, &name, &t, &businessKey, &started, &serviceName, &serviceMethod, &serviceType,
		&isForUpdate, &status, &inputParams, &outputParams, &errorBlob, &end, &stateIdCompensatedFor, &stateIdRetriedFor)
	if err != nil {
		return nil, err
	}

	if id.Valid {
		stateInstance.SetID(id.String)
	}
	if machineInstId.Valid {
		stateInstance.SetMachineInstanceID(machineInstId.String)
	}
	if name.Valid {
		stateInstance.SetName(name.String)
	}
	if t.Valid {
		stateInstance.SetType(t.String)
	}
	if businessKey.Valid {
		stateInstance.SetBusinessKey(businessKey.String)
	}
	if status.Valid {
		stateInstance.SetStatus(statelang.ExecutionStatus(status.String))
	}
	if started.Valid {
		startedTime, _ := time.Parse(TimeLayout, started.String)
		stateInstance.SetStartedTime(startedTime)
	}
	if end.Valid {
		endTime, _ := time.Parse(TimeLayout, end.String)
		stateInstance.SetEndTime(endTime)
	}

	if serviceName.Valid {
		stateInstance.SetServiceName(serviceName.String)
	}
	if serviceMethod.Valid {
		stateInstance.SetServiceMethod(serviceMethod.String)
	}
	if serviceType.Valid {
		stateInstance.SetServiceType(serviceType.String)
	}
	if isForUpdate.Valid {
		stateInstance.SetForUpdate(isForUpdate.Bool)
	}
	if stateIdCompensatedFor.Valid {
		stateInstance.SetStateIDCompensatedFor(stateIdCompensatedFor.String)
	}
	if stateIdRetriedFor.Valid {
		stateInstance.SetStateIDRetriedFor(stateIdRetriedFor.String)
	}

	if inputParams.Valid {
		stateInstance.SetSerializedInputParams(inputParams.String)
	}
	if outputParams.Valid {
		stateInstance.SetSerializedOutputParams(outputParams.String)
	}
	stateInstance.SetSerializedError(errorBlob)
	return stateInstance, err
}
