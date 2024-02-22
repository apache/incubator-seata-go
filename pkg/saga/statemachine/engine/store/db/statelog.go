package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/constant"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/process_ctrl"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/sequence"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine/serializer"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"github.com/seata/seata-go/pkg/util/log"
	"regexp"
	"strconv"
	"strings"
	"time"
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
		//TODO begin transaction
	}

	if machineInstance.ID() == "" && s.seqGenerator != nil {
		machineInstance.SetID(s.seqGenerator.GenerateId(constant.SeqEntityStateMachineInst, ""))
	}

	//TODO bind SAGA branch type

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

func (s *StateLogStore) RecordStateMachineFinished(ctx context.Context, machineInstance statelang.StateMachineInstance,
	context process_ctrl.ProcessContext) error {
	if machineInstance == nil {
		return nil
	}

	endParams := machineInstance.EndParams()

	if statelang.SU == machineInstance.Status() && machineInstance.Error() != nil {
		machineInstance.SetError(nil)
	}

	serializedEndParams, err := s.paramsSerializer.Serialize(endParams)
	if err != nil {
		return err
	}
	machineInstance.SetSerializedEndParams(serializedEndParams)
	serializedError, err := s.errorSerializer.Serialize(machineInstance.Error())
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

	//TODO check if timeout or else report transaction finished
	return nil
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
	isUpdateMode := s.isUpdateMode(stateInstance, context)

	if stateInstance.StateIDRetriedFor() != "" {
		if isUpdateMode {
			stateInstance.SetID(stateInstance.StateIDRetriedFor())
		} else {
			stateInstance.SetID(s.generateRetryStateInstanceId(stateInstance))
		}
	} else if stateInstance.StateIDCompensatedFor() != "" {
		stateInstance.SetID(s.generateCompensateStateInstanceId(stateInstance, isUpdateMode))
	} else {
		//TODO register branch
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

func (s *StateLogStore) isUpdateMode(instance statelang.StateInstance, context process_ctrl.ProcessContext) bool {
	//TODO implement me, add forward logic
	return false
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

	//TODO report branch
	return nil

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
		stateMachineInstance.SetError(deserializedError)
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
	//TODO add forward and compensate logic
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
