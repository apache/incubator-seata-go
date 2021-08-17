package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFuncIsExported(t *testing.T) {
	tests := []struct{
		name string
		funcName string
		expectedResult bool
	}{
		{
			name: "test func is exported",
			funcName: "Abb",
			expectedResult: true,
		},
		{
			name: "test func is not exported",
			funcName: "abb",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult := isExported(tt.funcName)
			assert.Equal(t,tt.expectedResult, actualResult)
		})
	}
}
