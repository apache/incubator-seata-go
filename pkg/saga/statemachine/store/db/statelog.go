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
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/config"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/pcext"
	"github.com/seata/seata-go/pkg/saga/statemachine/process_ctrl"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	constant2 "github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/protocol/branch"
	"github.com/seata/seata-go/pkg/protocol/message"
	"github.com/seata/seata-go/pkg/rm"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/serializer"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang/state"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/seata/seata-go/pkg/util/log"
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

	return nil
}

func (s *StateLogStore) beginTransaction(ctx context.Context, machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) error {
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

	tm.SetTxRole(ctx, tm.Launcher)
	tm.SetTxStatus(ctx, message.GlobalStatusUnKnown)
	tm.SetTxName(ctx, constant.SagaTransNamePrefix+machineInstance.StateMachine().Name())

	err := tm.GetGlobalTransactionManager().Begin(ctx, time.Duration(cfg.TransOperationTimeout()))
	if err != nil {
		return err
	}

	machineInstance.SetID(tm.GetXID(ctx))
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
		log.Warnf("StateMachineInstance[%s] is recovered by server, skip RecordStateMachineFinished", machineInstance.ID())
		return nil
	}

	// check if timeout or else report transaction finished
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(engine.StateMachineConfig)
	if !ok {
		return errors.New("stateMachineConfig is required in context")
	}

	if pcext.IsTimeout(machineInstance.UpdatedTime(), cfg.TransOperationTimeout()) {
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
	var err error
	defer func() {
		s.ClearUp(context)
		if err != nil {
			log.Errorf("Report transaction finish to server error: %v, StateMachine: %s, XID: %s, Reason: %s",
				err, machineInstance.StateMachine().Name(), machineInstance.ID(), err.Error())
		}
	}()

	globalTransaction, err := s.getGlobalTransaction(machineInstance, context)
	if err != nil {
		log.Errorf("Failed to get global transaction: %v", err)
		return err
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
	_, err = tm.GetGlobalTransactionManager().GlobalReport(ctx, globalTransaction)
	if err != nil {
		return err
	}
	return nil
}

func (s *StateLogStore) getGlobalTransaction(machineInstance statelang.StateMachineInstance, context process_ctrl.ProcessContext) (*tm.GlobalTransaction, error) {
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
	globalTransaction = &tm.GlobalTransaction{
		Xid:      xid,
		TxStatus: message.GlobalStatusUnKnown,
		TxRole:   tm.Launcher,
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
		s.branchRegister(stateInstance, context)
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
	return nil
}

func (s *StateLogStore) isUpdateMode(stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) (bool, error) {
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(config.DefaultStateMachineConfig)
	if !ok {
		return false, errors.New("stateMachineConfig is required in context")
	}

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
			return taskState.RetryPersistModeUpdate(), nil
		} else if stateMachine.IsRetryPersistModeUpdate() {
			return stateMachine.IsRetryPersistModeUpdate(), nil
		}
		return cfg.IsSagaRetryPersistModeUpdate(), nil
	} else if stateInstance.StateIDCompensatedFor() != "" {
		// find if this compensate has been executed
		stateList := stateInstance.StateMachineInstance().StateList()
		for _, instance := range stateList {
			if instance.IsForCompensation() && instance.Name() == stateInstance.Name() {
				if taskState != nil && taskState.CompensatePersistModeUpdate() {
					return taskState.CompensatePersistModeUpdate(), nil
				} else if stateMachine.IsCompensatePersistModeUpdate() {
					return stateMachine.IsCompensatePersistModeUpdate(), nil
				}
				return cfg.IsSagaCompensatePersistModeUpdate(), nil
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

func (s *StateLogStore) branchRegister(stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error {
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(config.DefaultStateMachineConfig)
	if !ok {
		return errors.New("stateMachineConfig is required in context")
	}

	if !cfg.IsSagaBranchRegisterEnable() {
		log.Debugf("sagaBranchRegisterEnable = false, skip branch report. state[%s]", stateInstance.Name())
		return nil
	}

	//Register branch
	var err error
	machineInstance := stateInstance.StateMachineInstance()
	defer func() {
		if err != nil {
			log.Errorf("Branch transaction failure. StateMachine: %s, XID: %s, State: %s, stateId: %s, err: %v",
				machineInstance.StateMachine().Name(), machineInstance.ID(), stateInstance.Name(), stateInstance.ID(), err)
		}
	}()

	globalTransaction, err := s.getGlobalTransaction(machineInstance, context)
	if err != nil {
		return err
	}
	if globalTransaction == nil {
		err = errors.New("Global transaction is not exists")
		return err
	}

	branchId, err := rm.GetRMRemotingInstance().BranchRegister(rm.BranchRegisterParam{
		BranchType: branch.BranchTypeSAGA,
		ResourceId: machineInstance.StateMachine().Name() + "#" + stateInstance.Name(),
		Xid:        globalTransaction.Xid,
	})
	if err != nil {
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

	_, err = ExecuteUpdate(s.db, s.recordStateFinishedSql, execStateInstanceStatementForUpdate, stateInstance)
	if err != nil {
		return err
	}

	// A switch to skip branch report on branch success, in order to optimize performance
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(config.DefaultStateMachineConfig)
	if !(ok && !cfg.IsRmReportSuccessEnable() && statelang.SU == stateInstance.Status()) {
		err = s.branchReport(stateInstance, context)
		return err
	}

	return nil

}

func (s *StateLogStore) branchReport(stateInstance statelang.StateInstance, context process_ctrl.ProcessContext) error {
	cfg, ok := context.GetVariable(constant.VarNameStateMachineConfig).(config.DefaultStateMachineConfig)
	if ok && !cfg.IsSagaBranchRegisterEnable() {
		log.Debugf("sagaBranchRegisterEnable = false, skip branch report. state[%s]", stateInstance.Name())
		return nil
	}

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

	globalTransaction, err := s.getGlobalTransaction(stateInstance.StateMachineInstance(), context)
	if err != nil {
		return err
	}
	if globalTransaction == nil {
		err = errors.New("Global transaction is not exists")
		return err
	}

	branchId, err := strconv.ParseInt(originalStateInst.ID(), 10, 0)
	err = rm.GetRMRemotingInstance().BranchReport(rm.BranchReportParam{
		BranchType: branch.BranchTypeSAGA,
		Xid:        globalTransaction.Xid,
		BranchId:   branchId,
		Status:     branchStatus,
	})
	return err
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

	result, err := stmt.Exec(
		obj.EndTime(),
		serializedError,
		obj.Status(),
		obj.SerializedOutputParams(),
		obj.EndTime(),
		obj.ID(),
		obj.MachineInstanceID(),
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
