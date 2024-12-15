package parser

import "testing"

func TestStateMachineConfigParser_Parse(t *testing.T) {
	parser := NewStateMachineConfigParser()

	tests := []struct {
		name           string
		configFilePath string
	}{
		{
			name:           "JSON Simple 1",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.json",
		},
		{
			name:           "JSON Simple 2",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.json",
		},
		{
			name:           "JSON Simple 3",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.json",
		},
		{
			name:           "YAML Simple 1",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.yaml",
		},
		{
			name:           "YAML Simple 2",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.yaml",
		},
		{
			name:           "YAML Simple 3",
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
