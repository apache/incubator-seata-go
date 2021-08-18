package proxy

import (
	"context"
	"reflect"
	"testing"

	"github.com/pkg/errors"
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

			assert.Equal(t, tt.expectedResult, actualResult)
		})
	}
}

func TestReturnWithError(t *testing.T) {
	testingErr := errors.New("testing err")

	tests := []struct{
		name string
		methodDesc *MethodDescriptor
		err error
		expectedResult []reflect.Value
	}{
		{
			name: "test return with error when methodDesc.ReturnValuesNum is 2",
			methodDesc: &MethodDescriptor{
				ReturnValuesNum: 2,
				ReturnValuesType: []reflect.Type{
					reflect.TypeOf("1"),
					reflect.TypeOf(errors.New("method output error")),
				},
			},
			err: testingErr,
			expectedResult: []reflect.Value{
				reflect.Zero(reflect.TypeOf("1")),
				reflect.ValueOf(testingErr),
			},
		},
		{
			name: "test return with error when methodDesc.ReturnValuesNum is 1",
			methodDesc: &MethodDescriptor{
				ReturnValuesNum: 1,
				ReturnValuesType: []reflect.Type{
					reflect.TypeOf(errors.New("method output error")),
				},
			},
			err: testingErr,
			expectedResult: []reflect.Value{
				reflect.ValueOf(testingErr),
			},
		},
		{
			name: "test return with error when methodDesc.ReturnValuesNum is 0",
			methodDesc: &MethodDescriptor{
				ReturnValuesNum: 0,
				ReturnValuesType: []reflect.Type{
				},
			},
			err: testingErr,
			expectedResult: []reflect.Value{
				reflect.ValueOf(testingErr),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult := ReturnWithError(tt.methodDesc, tt.err)

			assert.Equal(t, tt.expectedResult, actualResult)
		})
	}
}
