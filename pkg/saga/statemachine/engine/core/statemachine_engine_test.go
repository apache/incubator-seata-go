package core

import (
	"context"
	"testing"
)

func TestEngine(t *testing.T) {

}

func TestSimpleStateMachine(t *testing.T) {
	engine := NewProcessCtrlStateMachineEngine()
	engine.Start(context.Background(), "simpleStateMachine", "tenantId", nil)
}
