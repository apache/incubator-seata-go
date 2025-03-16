package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		if test.expectError {
			assert.Error(t, err, "Expected an error for input '%s'", test.expressionStr)
		} else {
			assert.NoError(t, err, "Did not expect an error for input '%s'", test.expressionStr)
			assert.NotNil(t, result, "Expected a non-nil result for input '%s'", test.expressionStr)
			assert.Equal(t, *test.expected, *result, "Expected result %+v, got %+v for input '%s'", test.expected, result, test.expressionStr)
		}
	}
}
