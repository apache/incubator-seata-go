package expr

import (
	"fmt"
	"testing"
)

func TestParseExpressionStruct(t *testing.T) {
	tests := []struct {
		expressionStr string
		expected      *ExpressionStruct
		expectError   bool
	}{
		{
			expressionStr: "$type{content}",
			expected: &ExpressionStruct{
				typeStart: 0,
				typeEnd:   5,
				typ:       "type",
				end:       13,
				content:   "content",
			},
			expectError: false,
		},
		{
			expressionStr: "$type.content",
			expected: &ExpressionStruct{
				typeStart: 0,
				typeEnd:   5,
				typ:       "type",
				end:       13,
				content:   "content",
			},
			expectError: false,
		},
		{
			expressionStr: "invalid expression",
			expected:      nil,
			expectError:   true,
		},
	}

	for _, test := range tests {
		result, err := parseExpressionStruct(test.expressionStr)

		if (err != nil) != test.expectError {
			fmt.Printf("result: %#v", result)
			fmt.Println("err:", err)
			fmt.Println("err != nil:", err != nil)
			t.Errorf("For input '%s', expected error: %v, got: %v", test.expressionStr, test.expectError, err)
		}

		if !test.expectError && result != nil {
			if *result != *test.expected {
				t.Errorf("For input '%s', expected: %+v, got: %+v", test.expressionStr, test.expected, result)
			}
		}
	}
}
