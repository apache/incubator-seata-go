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

package constant

type NetExceptionType string

const (
	// region State Types
	StateTypeServiceTask          string = "ServiceTask"
	StateTypeChoice               string = "Choice"
	StateTypeFail                 string = "Fail"
	StateTypeSucceed              string = "Succeed"
	StateTypeCompensationTrigger  string = "CompensationTrigger"
	StateTypeSubStateMachine      string = "SubStateMachine"
	StateTypeCompensateSubMachine string = "CompensateSubMachine"
	StateTypeScriptTask           string = "ScriptTask"
	StateTypeLoopStart            string = "LoopStart"
	// end region

	// region Service Types
	ServiceTypeGRPC string = "GRPC"
	// end region

	// region System Variables
	VarNameOutputParams                  string = "outputParams"
	VarNameProcessType                   string = "_ProcessType_"
	VarNameOperationName                 string = "_operation_name_"
	OperationNameStart                   string = "start"
	OperationNameCompensate              string = "compensate"
	VarNameAsyncCallback                 string = "_async_callback_"
	VarNameCurrentExceptionRoute         string = "_current_exception_route_"
	VarNameIsExceptionNotCatch           string = "_is_exception_not_catch_"
	VarNameSubMachineParentId            string = "_sub_machine_parent_id_"
	VarNameCurrentChoice                 string = "_current_choice_"
	VarNameStateMachineInst              string = "_current_statemachine_instance_"
	VarNameStateMachine                  string = "_current_statemachine_"
	VarNameStateMachineEngine            string = "_current_statemachine_engine_"
	VarNameStateMachineConfig            string = "_statemachine_config_"
	VarNameStateMachineContext           string = "context"
	VarNameIsAsyncExecution              string = "_is_async_execution_"
	VarNameStateInst                     string = "_current_state_instance_"
	VarNameBusinesskey                   string = "_business_key_"
	VarNameParentId                      string = "_parent_id_"
	VarNameCurrentException              string = "currentException"
	CompensateSubMachineStateNamePrefix  string = "_compensate_sub_machine_state_"
	DefaultScriptType                    string = "groovy"
	VarNameSyncExeStack                  string = "_sync_execution_stack_"
	VarNameInputParams                   string = "inputParams"
	VarNameIsLoopState                   string = "_is_loop_state_"
	VarNameCurrentCompensateTriggerState string = "_is_compensating_"
	VarNameCurrentCompensationHolder     string = "_current_compensation_holder_"
	VarNameFirstCompensationStateStarted string = "_first_compensation_state_started"
	VarNameCurrentLoopContextHolder      string = "_current_loop_context_holder_"
	VarNameRetriedStateInstId            string = "_retried_state_instance_id"
	VarNameIsForSubStatMachineForward    string = "_is_for_sub_statemachine_forward_"
	// TODO: this lock in process context only has one, try to add more to add concurrent
	VarNameProcessContextMutexLock string = "_current_context_mutex_lock"
	VarNameFailEndStateFlag        string = "_fail_end_state_flag_"
	VarNameGlobalTx                string = "_global_transaction_"
	// end region

	// region of loop
	LoopCounter                string = "loopCounter"
	LoopSemaphore              string = "loopSemaphore"
	LoopResult                 string = "loopResult"
	NumberOfInstances          string = "nrOfInstances"
	NumberOfActiveInstances    string = "nrOfActiveInstances"
	NumberOfCompletedInstances string = "nrOfCompletedInstances"
	// end region

	// region others
	SeqEntityStateMachine     string = "STATE_MACHINE"
	SeqEntityStateMachineInst string = "STATE_MACHINE_INST"
	SeqEntityStateInst        string = "STATE_INST"
	OperationNameForward      string = "forward"
	LoopStateNamePattern      string = "-loop-"
	SagaTransNamePrefix       string = "$Saga_"
	// end region

	SeperatorParentId string = ":"

	// Machine execution timeout error code
	FrameworkErrorCodeStateMachineExecutionTimeout                  = "StateMachineExecutionTimeout"
	ConnectException                               NetExceptionType = "CONNECT_EXCEPTION"
	ConnectTimeoutException                        NetExceptionType = "CONNECT_TIMEOUT_EXCEPTION"
	ReadTimeoutException                           NetExceptionType = "READ_TIMEOUT_EXCEPTION"
	NotNetException                                NetExceptionType = "NOT_NET_EXCEPTION"
)
