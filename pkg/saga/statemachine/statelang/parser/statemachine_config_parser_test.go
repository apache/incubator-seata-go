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

package parser

import (
	"github.com/seata/seata-go/pkg/saga/statemachine"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStateMachineConfigParser_Parse(t *testing.T) {
	parser := NewStateMachineConfigParser()

	tests := []struct {
		name           string
		configFilePath string
		expectedObject *statemachine.StateMachineObject
	}{
		{
			name:           "JSON Simple 1",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.json",
			expectedObject: GetStateMachineObject1("json"),
		},
		{
			name:           "JSON Simple 2",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.json",
			expectedObject: GetStateMachineObject2("json"),
		},
		{
			name:           "JSON Simple 3",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.json",
			expectedObject: GetStateMachineObject3("json"),
		},
		{
			name:           "YAML Simple 1",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statelang_with_choice.yaml",
			expectedObject: GetStateMachineObject1("yaml"),
		},
		{
			name:           "YAML Simple 2",
			configFilePath: "../../../../../testdata/saga/statelang/simple_statemachine.yaml",
			expectedObject: GetStateMachineObject2("yaml"),
		},
		{
			name:           "YAML Simple 3",
			configFilePath: "../../../../../testdata/saga/statelang/state_machine_new_designer.yaml",
			expectedObject: GetStateMachineObject3("yaml"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := parser.ReadConfigFile(tt.configFilePath)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
			object, err := parser.Parse(content)
			if err != nil {
				t.Error("parse fail: " + err.Error())
			}
			assert.Equal(t, tt.expectedObject, object)
		})
	}
}

func GetStateMachineObject1(format string) *statemachine.StateMachineObject {
	switch format {
	case "json":
	case "yaml":
	}

	return &statemachine.StateMachineObject{
		Name:       "simpleChoiceTestStateMachine",
		Comment:    "带条件分支的测试状态机定义",
		StartState: "FirstState",
		Version:    "0.0.1",
		States: map[string]interface{}{
			"FirstState": map[string]interface{}{
				"Type":          "ServiceTask",
				"ServiceName":   "demoService",
				"ServiceMethod": "foo",
				"Next":          "ChoiceState",
			},
			"ChoiceState": map[string]interface{}{
				"Type": "Choice",
				"Choices": []interface{}{
					map[string]interface{}{
						"Expression": "[a] == 1",
						"Next":       "SecondState",
					},
					map[string]interface{}{
						"Expression": "[a] == 2",
						"Next":       "ThirdState",
					},
				},
				"Default": "SecondState",
			},
			"SecondState": map[string]interface{}{
				"Type":          "ServiceTask",
				"ServiceName":   "demoService",
				"ServiceMethod": "bar",
			},
			"ThirdState": map[string]interface{}{
				"Type":          "ServiceTask",
				"ServiceName":   "demoService",
				"ServiceMethod": "foo",
			},
		},
	}
}

func GetStateMachineObject2(format string) *statemachine.StateMachineObject {
	var retryMap map[string]interface{}

	switch format {
	case "json":
		retryMap = map[string]interface{}{
			"Exceptions": []interface{}{
				"java.lang.Exception",
			},
			"IntervalSeconds": float64(2),
			"MaxAttempts":     float64(3),
			"BackoffRate":     1.5,
		}
	case "yaml":
		retryMap = map[string]interface{}{
			"Exceptions": []interface{}{
				"java.lang.Exception",
			},
			"IntervalSeconds": 2,
			"MaxAttempts":     3,
			"BackoffRate":     1.5,
		}
	}

	return &statemachine.StateMachineObject{
		Name:       "simpleTestStateMachine",
		Comment:    "测试状态机定义",
		StartState: "FirstState",
		Version:    "0.0.1",
		States: map[string]interface{}{
			"FirstState": map[string]interface{}{
				"Type":          "ServiceTask",
				"ServiceName":   "is.seata.saga.DemoService",
				"ServiceMethod": "foo",
				"IsPersist":     false,
				"Next":          "ScriptState",
			},
			"ScriptState": map[string]interface{}{
				"Type":          "ScriptTask",
				"ScriptType":    "groovy",
				"ScriptContent": "return 'hello ' + inputA",
				"Input": []interface{}{
					map[string]interface{}{
						"inputA": "$.data1",
					},
				},
				"Output": map[string]interface{}{
					"scriptStateResult": "$.#root",
				},
				"Next": "ChoiceState",
			},
			"ChoiceState": map[string]interface{}{
				"Type": "Choice",
				"Choices": []interface{}{
					map[string]interface{}{
						"Expression": "foo == 1",
						"Next":       "FirstMatchState",
					},
					map[string]interface{}{
						"Expression": "foo == 2",
						"Next":       "SecondMatchState",
					},
				},
				"Default": "FailState",
			},
			"FirstMatchState": map[string]interface{}{
				"Type":            "ServiceTask",
				"ServiceName":     "is.seata.saga.DemoService",
				"ServiceMethod":   "bar",
				"CompensateState": "CompensateFirst",
				"Status": map[string]interface{}{
					"return.code == 'S'":              "SU",
					"return.code == 'F'":              "FA",
					"$exception{java.lang.Throwable}": "UN",
				},
				"Input": []interface{}{
					map[string]interface{}{
						"inputA1": "$.data1",
						"inputA2": map[string]interface{}{
							"a": "$.data2.a",
						},
					},
					map[string]interface{}{
						"inputB": "$.header",
					},
				},
				"Output": map[string]interface{}{
					"firstMatchStateResult": "$.#root",
				},
				"Retry": []interface{}{
					retryMap,
				},
				"Catch": []interface{}{
					map[string]interface{}{
						"Exceptions": []interface{}{
							"java.lang.Exception",
						},
						"Next": "CompensationTrigger",
					},
				},
				"Next": "SuccessState",
			},
			"CompensateFirst": map[string]interface{}{
				"Type":              "ServiceTask",
				"ServiceName":       "is.seata.saga.DemoService",
				"ServiceMethod":     "compensateBar",
				"IsForCompensation": true,
				"IsForUpdate":       true,
				"Input": []interface{}{
					map[string]interface{}{
						"input": "$.data",
					},
				},
				"Output": map[string]interface{}{
					"firstMatchStateResult": "$.#root",
				},
				"Status": map[string]interface{}{
					"return.code == 'S'":              "SU",
					"return.code == 'F'":              "FA",
					"$exception{java.lang.Throwable}": "UN",
				},
			},
			"CompensationTrigger": map[string]interface{}{
				"Type": "CompensationTrigger",
				"Next": "CompensateEndState",
			},
			"CompensateEndState": map[string]interface{}{
				"Type":      "Fail",
				"ErrorCode": "StateCompensated",
				"Message":   "State Compensated!",
			},
			"SecondMatchState": map[string]interface{}{
				"Type":             "SubStateMachine",
				"StateMachineName": "simpleTestSubStateMachine",
				"Input": []interface{}{
					map[string]interface{}{
						"input": "$.data",
					},
					map[string]interface{}{
						"header": "$.header",
					},
				},
				"Output": map[string]interface{}{
					"firstMatchStateResult": "$.#root",
				},
				"Next": "SuccessState",
			},
			"FailState": map[string]interface{}{
				"Type":      "Fail",
				"ErrorCode": "DefaultStateError",
				"Message":   "No Matches!",
			},
			"SuccessState": map[string]interface{}{
				"Type": "Succeed",
			},
		},
	}
}

func GetStateMachineObject3(format string) *statemachine.StateMachineObject {
	var (
		boundsMap1 map[string]interface{}
		boundsMap2 map[string]interface{}
		boundsMap3 map[string]interface{}
		boundsMap4 map[string]interface{}
		boundsMap5 map[string]interface{}
		boundsMap6 map[string]interface{}
		boundsMap7 map[string]interface{}
		boundsMap8 map[string]interface{}
		boundsMap9 map[string]interface{}

		waypoints1 []interface{}
		waypoints2 []interface{}
		waypoints3 []interface{}
		waypoints4 []interface{}
		waypoints5 []interface{}
		waypoints6 []interface{}
		waypoints7 []interface{}
	)

	switch format {
	case "json":
		boundsMap1 = map[string]interface{}{
			"x":      float64(300),
			"y":      float64(178),
			"width":  float64(100),
			"height": float64(80),
		}
		boundsMap2 = map[string]interface{}{
			"x":      float64(455),
			"y":      float64(193),
			"width":  float64(50),
			"height": float64(50),
		}
		boundsMap3 = map[string]interface{}{
			"x":      float64(300),
			"y":      float64(310),
			"width":  float64(100),
			"height": float64(80),
		}
		boundsMap4 = map[string]interface{}{
			"x":      float64(550),
			"y":      float64(178),
			"width":  float64(100),
			"height": float64(80),
		}
		boundsMap5 = map[string]interface{}{
			"x":      float64(550),
			"y":      float64(310),
			"width":  float64(100),
			"height": float64(80),
		}
		boundsMap6 = map[string]interface{}{
			"x":      float64(632),
			"y":      float64(372),
			"width":  float64(36),
			"height": float64(36),
		}
		boundsMap7 = map[string]interface{}{
			"x":      float64(722),
			"y":      float64(200),
			"width":  float64(36),
			"height": float64(36),
		}
		boundsMap8 = map[string]interface{}{
			"x":      float64(722),
			"y":      float64(372),
			"width":  float64(36),
			"height": float64(36),
		}
		boundsMap9 = map[string]interface{}{
			"x":      float64(812),
			"y":      float64(372),
			"width":  float64(36),
			"height": float64(36),
		}

		waypoints1 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(400),
					"y": float64(218),
				},
				"x": float64(400),
				"y": float64(218),
			},
			map[string]interface{}{"x": float64(435), "y": float64(218)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(455),
					"y": float64(218),
				},
				"x": float64(455),
				"y": float64(218),
			},
		}
		waypoints2 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(505),
					"y": float64(218),
				},
				"x": float64(505),
				"y": float64(218),
			},
			map[string]interface{}{"x": float64(530), "y": float64(218)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(550),
					"y": float64(218),
				},
				"x": float64(550),
				"y": float64(218),
			},
		}
		waypoints3 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(480),
					"y": float64(243),
				},
				"x": float64(480),
				"y": float64(243),
			},
			map[string]interface{}{"x": float64(600), "y": float64(290)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(600),
					"y": float64(310),
				},
				"x": float64(600),
				"y": float64(310),
			},
		}
		waypoints4 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(650),
					"y": float64(218),
				},
				"x": float64(650),
				"y": float64(218),
			},
			map[string]interface{}{"x": float64(702), "y": float64(218)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(722),
					"y": float64(218),
				},
				"x": float64(722),
				"y": float64(218),
			},
		}
		waypoints5 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(668),
					"y": float64(390),
				},
				"x": float64(668),
				"y": float64(390),
			},
			map[string]interface{}{"x": float64(702), "y": float64(390)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(722),
					"y": float64(390),
				},
				"x": float64(722),
				"y": float64(390),
			},
		}
		waypoints6 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(600),
					"y": float64(310),
				},
				"x": float64(600),
				"y": float64(310),
			},
			map[string]interface{}{"x": float64(740), "y": float64(256)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(740),
					"y": float64(236),
				},
				"x": float64(740),
				"y": float64(236),
			},
		}
		waypoints7 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(758),
					"y": float64(390),
				},
				"x": float64(758),
				"y": float64(390),
			},
			map[string]interface{}{"x": float64(792), "y": float64(390)},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": float64(812),
					"y": float64(390),
				},
				"x": float64(812),
				"y": float64(390),
			},
		}

	case "yaml":
		boundsMap1 = map[string]interface{}{
			"x":      300,
			"y":      178,
			"width":  100,
			"height": 80,
		}
		boundsMap2 = map[string]interface{}{
			"x":      455,
			"y":      193,
			"width":  50,
			"height": 50,
		}
		boundsMap3 = map[string]interface{}{
			"x":      300,
			"y":      310,
			"width":  100,
			"height": 80,
		}
		boundsMap4 = map[string]interface{}{
			"x":      550,
			"y":      178,
			"width":  100,
			"height": 80,
		}
		boundsMap5 = map[string]interface{}{
			"x":      550,
			"y":      310,
			"width":  100,
			"height": 80,
		}
		boundsMap6 = map[string]interface{}{
			"x":      632,
			"y":      372,
			"width":  36,
			"height": 36,
		}
		boundsMap7 = map[string]interface{}{
			"x":      722,
			"y":      200,
			"width":  36,
			"height": 36,
		}
		boundsMap8 = map[string]interface{}{
			"x":      722,
			"y":      372,
			"width":  36,
			"height": 36,
		}
		boundsMap9 = map[string]interface{}{
			"x":      812,
			"y":      372,
			"width":  36,
			"height": 36,
		}

		waypoints1 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 400,
					"y": 218,
				},
				"x": 400,
				"y": 218,
			},
			map[string]interface{}{"x": 435, "y": 218},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 455,
					"y": 218,
				},
				"x": 455,
				"y": 218,
			},
		}
		waypoints2 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 505,
					"y": 218,
				},
				"x": 505,
				"y": 218,
			},
			map[string]interface{}{"x": 530, "y": 218},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 550,
					"y": 218,
				},
				"x": 550,
				"y": 218,
			},
		}
		waypoints3 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 480,
					"y": 243,
				},
				"x": 480,
				"y": 243,
			},
			map[string]interface{}{"x": 600, "y": 290},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 600,
					"y": 310,
				},
				"x": 600,
				"y": 310,
			},
		}
		waypoints4 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 650,
					"y": 218,
				},
				"x": 650,
				"y": 218,
			},
			map[string]interface{}{"x": 702, "y": 218},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 722,
					"y": 218,
				},
				"x": 722,
				"y": 218,
			},
		}
		waypoints5 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 668,
					"y": 390,
				},
				"x": 668,
				"y": 390,
			},
			map[string]interface{}{"x": 702, "y": 390},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 722,
					"y": 390,
				},
				"x": 722,
				"y": 390,
			},
		}
		waypoints6 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 600,
					"y": 310,
				},
				"x": 600,
				"y": 310,
			},
			map[string]interface{}{"x": 740, "y": 256},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 740,
					"y": 236,
				},
				"x": 740,
				"y": 236,
			},
		}
		waypoints7 = []interface{}{
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 758,
					"y": 390,
				},
				"x": 758,
				"y": 390,
			},
			map[string]interface{}{"x": 792, "y": 390},
			map[string]interface{}{
				"original": map[string]interface{}{
					"x": 812,
					"y": 390,
				},
				"x": 812,
				"y": 390,
			},
		}
	}

	return &statemachine.StateMachineObject{
		Name:                        "StateMachineNewDesigner",
		Comment:                     "This state machine is modeled by designer tools.",
		Version:                     "0.0.1",
		StartState:                  "ServiceTask-a9h2o51",
		RecoverStrategy:             "",
		Persist:                     false,
		RetryPersistModeUpdate:      false,
		CompensatePersistModeUpdate: false,
		Type:                        "",
		States: map[string]interface{}{
			"ServiceTask-a9h2o51": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap1,
				},
				"Name":              "ServiceTask-a9h2o51",
				"IsForCompensation": false,
				"Input":             []interface{}{map[string]interface{}{}},
				"Output":            map[string]interface{}{},
				"Status":            map[string]interface{}{},
				"Retry":             []interface{}{},
				"ServiceName":       "",
				"ServiceMethod":     "",
				"Type":              "ServiceTask",
				"Next":              "Choice-4ajl8nt",
				"edge": map[string]interface{}{
					"Choice-4ajl8nt": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints1,
							"source":    "ServiceTask-a9h2o51",
							"target":    "Choice-4ajl8nt",
						},
						"Type": "Transition",
					},
				},
				"CompensateState": "CompensateFirstState",
			},
			"Choice-4ajl8nt": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap2,
				},
				"Name": "Choice-4ajl8nt",
				"Type": "Choice",
				"Choices": []interface{}{
					map[string]interface{}{
						"Expression": "",
						"Next":       "SubStateMachine-cauj9uy",
					},
					map[string]interface{}{
						"Expression": "",
						"Next":       "ServiceTask-vdij28l",
					},
				},
				"Default": "SubStateMachine-cauj9uy",
				"edge": map[string]interface{}{
					"SubStateMachine-cauj9uy": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints2,
							"source":    "Choice-4ajl8nt",
							"target":    "SubStateMachine-cauj9uy",
						},
						"Type": "ChoiceEntry",
					},
					"ServiceTask-vdij28l": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints3,
							"source":    "Choice-4ajl8nt",
							"target":    "ServiceTask-vdij28l",
						},
						"Type": "ChoiceEntry",
					},
				},
			},
			"CompensateFirstState": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap3,
				},
				"Name":              "CompensateFirstState",
				"IsForCompensation": true,
				"Input":             []interface{}{map[string]interface{}{}},
				"Output":            map[string]interface{}{},
				"Status":            map[string]interface{}{},
				"Retry":             []interface{}{},
				"ServiceName":       "",
				"ServiceMethod":     "",
				"Type":              "ServiceTask",
			},
			"SubStateMachine-cauj9uy": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap4,
				},
				"Name":              "SubStateMachine-cauj9uy",
				"IsForCompensation": false,
				"Input":             []interface{}{map[string]interface{}{}},
				"Output":            map[string]interface{}{},
				"Status":            map[string]interface{}{},
				"Retry":             []interface{}{},
				"StateMachineName":  "",
				"Type":              "SubStateMachine",
				"Next":              "Succeed-5x3z98u",
				"edge": map[string]interface{}{
					"Succeed-5x3z98u": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints4,
							"source":    "SubStateMachine-cauj9uy",
							"target":    "Succeed-5x3z98u",
						},
						"Type": "Transition",
					},
				},
			},
			"ServiceTask-vdij28l": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap5,
				},
				"Name":              "ServiceTask-vdij28l",
				"IsForCompensation": false,
				"Input":             []interface{}{map[string]interface{}{}},
				"Output":            map[string]interface{}{},
				"Status":            map[string]interface{}{},
				"Retry":             []interface{}{},
				"ServiceName":       "",
				"ServiceMethod":     "",
				"Catch": []interface{}{
					map[string]interface{}{
						"Exceptions": []interface{}{},
						"Next":       "CompensationTrigger-uldp2ou",
					},
				},
				"Type": "ServiceTask",
				"catch": map[string]interface{}{
					"style": map[string]interface{}{
						"bounds": boundsMap6,
					},
					"edge": map[string]interface{}{
						"CompensationTrigger-uldp2ou": map[string]interface{}{
							"style": map[string]interface{}{
								"waypoints": waypoints5,
								"source":    "ServiceTask-vdij28l",
								"target":    "CompensationTrigger-uldp2ou",
							},
							"Type": "ExceptionMatch",
						},
					},
				},
				"Next": "Succeed-5x3z98u",
				"edge": map[string]interface{}{
					"Succeed-5x3z98u": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints6,
							"source":    "ServiceTask-vdij28l",
							"target":    "Succeed-5x3z98u",
						},
						"Type": "Transition",
					},
				},
			},
			"Succeed-5x3z98u": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap7,
				},
				"Name": "Succeed-5x3z98u",
				"Type": "Succeed",
			},
			"CompensationTrigger-uldp2ou": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap8,
				},
				"Name": "CompensationTrigger-uldp2ou",
				"Type": "CompensationTrigger",
				"Next": "Fail-9roxcv5",
				"edge": map[string]interface{}{
					"Fail-9roxcv5": map[string]interface{}{
						"style": map[string]interface{}{
							"waypoints": waypoints7,
							"source":    "CompensationTrigger-uldp2ou",
							"target":    "Fail-9roxcv5",
						},
						"Type": "Transition",
					},
				},
			},
			"Fail-9roxcv5": map[string]interface{}{
				"style": map[string]interface{}{
					"bounds": boundsMap9,
				},
				"Name":      "Fail-9roxcv5",
				"ErrorCode": "",
				"Message":   "",
				"Type":      "Fail",
			},
		},
	}
}
