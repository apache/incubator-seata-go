package proxy

import (
	"context"
	"reflect"
	"testing"

	"github.com/opentrx/seata-golang/v2/pkg/util/log"
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

func TestSuiteContext(t *testing.T) {
	emptyContext := context.TODO()
	validContext := context.WithValue(context.TODO(), "aa", "bb")
	tests := []struct{
		name string
		methodDesc *MethodDescriptor
		ctx context.Context
		expectedResult reflect.Value
	}{
		{
			name: "test suite valid context",
			methodDesc: &MethodDescriptor{},
			ctx: emptyContext,
			expectedResult: reflect.ValueOf(emptyContext),
		},
		{
			name: "test suite invalid context",
			methodDesc: &MethodDescriptor{
				CtxType: reflect.TypeOf(validContext),
			},
			ctx: nil,
			expectedResult: reflect.Zero(reflect.TypeOf(validContext)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult := SuiteContext(tt.methodDesc, tt.ctx)

			log.Debugf("expectedResult: %+v", tt.expectedResult)
			log.Debugf("actualResult: %+v", actualResult)

			assert.Equal(t, tt.expectedResult, actualResult)
		})
	}
}
