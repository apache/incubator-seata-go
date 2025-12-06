package pcext_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"seata.apache.org/seata-go/pkg/saga/statemachine/constant"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/config"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/pcext"
	"seata.apache.org/seata-go/pkg/saga/statemachine/engine/utils"
	"seata.apache.org/seata-go/pkg/saga/statemachine/process_ctrl/process"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang"
	"seata.apache.org/seata-go/pkg/saga/statemachine/statelang/parser"
	stateimpl "seata.apache.org/seata-go/pkg/saga/statemachine/statelang/state"
)

const compensationRouteMachine = `{
  "Name": "CompRouteTest",
  "StartState": "ServiceB",
  "States": {
    "ServiceA": {
      "Type": "ServiceTask",
      "ServiceType": "local",
      "ServiceName": "noopService",
      "ServiceMethod": "Do",
      "CompensateState": "CompensateA",
      "ForCompensation": false,
      "ForUpdate": false,
      "Next": "Fail"
    },
    "ServiceB": {
      "Type": "ServiceTask",
      "ServiceType": "local",
      "ServiceName": "noopService",
      "ServiceMethod": "Do",
      "CompensateState": "CompensateB",
      "ForCompensation": false,
      "ForUpdate": false,
      "Next": "Fail"
    },
    "CompensateA": {
      "Type": "ServiceTask",
      "ServiceType": "local",
      "ServiceName": "noopService",
      "ServiceMethod": "Do",
      "ForCompensation": true
    },
    "CompensateB": {
      "Type": "ServiceTask",
      "ServiceType": "local",
      "ServiceName": "noopService",
      "ServiceMethod": "Do",
      "ForCompensation": true
    },
    "Fail": {
      "Type": "Fail"
    }
  }
}`

func TestTaskStateRouterBuildsCompensationStackFromExecutedStates(t *testing.T) {
	cfg, err := config.NewDefaultStateMachineConfig(
		config.WithEnableAsync(false),
		config.WithStateMachineResources(nil),
	)
	require.NoError(t, err)

	sm, err := parser.NewJSONStateMachineParser().Parse(compensationRouteMachine)
	require.NoError(t, err)
	sm.SetTenantId(cfg.GetDefaultTenantId())
	sm.SetContent(compensationRouteMachine)
	require.NoError(t, cfg.StateMachineRepository().RegistryStateMachine(sm))

	inst := statelang.NewStateMachineInstanceImpl()
	inst.SetID("comp-route-inst")
	inst.SetStateMachine(sm)
	inst.SetStatus(statelang.RU)
	inst.SetCompensationStatus("")

	serviceAInst := statelang.NewStateInstanceImpl()
	serviceAInst.SetID("svcA-1")
	serviceAInst.SetName("ServiceA")
	serviceAInst.SetType(constant.StateTypeServiceTask)
	serviceAInst.SetMachineInstanceID(inst.ID())
	serviceAInst.SetStateMachineInstance(inst)
	serviceAInst.SetStatus(statelang.RU)

	serviceBInst := statelang.NewStateInstanceImpl()
	serviceBInst.SetID("svcB-1")
	serviceBInst.SetName("ServiceB")
	serviceBInst.SetType(constant.StateTypeServiceTask)
	serviceBInst.SetMachineInstanceID(inst.ID())
	serviceBInst.SetStateMachineInstance(inst)
	serviceBInst.SetStatus(statelang.SU)

	inst.PutState(serviceAInst.ID(), serviceAInst)
	inst.PutState(serviceBInst.ID(), serviceBInst)

	ctx := utils.NewProcessContextBuilder().
		WithProcessType(process.StateLang).
		WithOperationName(constant.OperationNameForward).
		WithInstruction(pcext.NewStateInstruction(sm.Name(), sm.StartState())).
		WithStateMachineInstance(inst).
		Build()

	ctx.SetVariable(constant.VarNameStateMachineInst, inst)
	ctx.SetVariable(constant.VarNameStateMachine, sm)
	ctx.SetVariable(constant.VarNameStateMachineConfig, cfg)
	ctx.SetVariable(constant.VarNameStateInst, statelang.StateInstance(nil))

	router := pcext.TaskStateRouter{}
	compTrigger := stateimpl.NewCompensationTriggerStateImpl()
	compTrigger.SetStateMachine(sm)

	_, err = router.Route(context.Background(), ctx, compTrigger)
	require.NoError(t, err)

	holder := pcext.GetCurrentCompensationHolder(context.Background(), ctx, true)
	require.NotNil(t, holder)

	compensated := make(map[string]string)
	holder.StatesNeedCompensation().Range(func(key, value any) bool {
		stateName, ok := key.(string)
		require.True(t, ok)
		inst, ok := value.(statelang.StateInstance)
		require.True(t, ok)
		compensated[stateName] = inst.Name()
		return true
	})

	require.Len(t, compensated, 1)
	require.Equal(t, "ServiceB", compensated["CompensateB"])
}
