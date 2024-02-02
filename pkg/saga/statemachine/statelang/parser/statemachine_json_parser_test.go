package parser

import (
	"os"
	"testing"
)

func TestParseChoice(t *testing.T) {
	filePath := "../../../../../testdata/saga/statelang/simple_statelang_with_choice.json"
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Error("parse fail: " + err.Error())
		return
	}
	_, err = NewJSONStateMachineParser().Parse(string(fileContent))
	if err != nil {
		t.Error("parse fail: " + err.Error())
	}
}

func TestParseServiceTaskForSimpleStateMachine(t *testing.T) {
	filePath := "../../../../../testdata/saga/statelang/simple_statemachine.json"
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Error("parse fail: " + err.Error())
		return
	}
	_, err = NewJSONStateMachineParser().Parse(string(fileContent))
	if err != nil {
		t.Error("parse fail: " + err.Error())
	}
}

func TestParseServiceTaskForNewDesigner(t *testing.T) {
	filePath := "../../../../../testdata/saga/statelang/state_machine_new_designer.json"
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Error("parse fail: " + err.Error())
		return
	}
	_, err = NewJSONStateMachineParser().Parse(string(fileContent))
	if err != nil {
		t.Error("parse fail: " + err.Error())
	}
}
