package parser

import (
	"testing"
)

func TestParseChoice(t *testing.T) {
	var content = "{\n    \"Name\":\"ChoiceTest\",\n    \"Comment\":\"ChoiceTest\",\n    \"StartState\":\"ChoiceState\",\n    \"Version\":\"0.0.1\",\n    \"States\":{\n        \"ChoiceState\":{\n            \"Type\":\"Choice\",\n            \"Choices\":[\n                {\n                    \"Expression\":\"[a] == 1\",\n                    \"Next\":\"SecondState\"\n                },\n                {\n                    \"Expression\":\"[a] == 2\",\n                    \"Next\":\"ThirdState\"\n                }\n            ],\n            \"Default\":\"Fail\"\n        }\n    }\n}"
	_, err := NewJSONStateMachineParser().Parse(content)
	if err != nil {
		t.Error("parse fail: " + err.Error())
	}
}
