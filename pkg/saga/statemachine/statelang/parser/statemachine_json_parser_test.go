package parser

import (
	"testing"
)

func TestParseChoice(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateLang With Choice",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.json",
		},
		{
			name:           "YAML Simple: StateLang With Choice",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.configFilePath)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}

func TestParseServiceTaskForSimpleStateMachine(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateMachine",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.json",
		},
		{
			name:           "YAML Simple: StateMachine",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.configFilePath)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}

func TestParseServiceTaskForNewDesigner(t *testing.T) {
	parser := NewJSONStateMachineParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple: StateMachine New Designer",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.json",
		},
		{
			name:           "YAML Simple: StateMachine New Designer",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.configFilePath)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
		})
	}
}
