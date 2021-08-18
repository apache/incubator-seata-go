package proxy

import (
	"context"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsExportedOrBuiltinType(t *testing.T) {
	type ExportedStruct struct {
		Abb string
	}
	type notExportedStruct struct {
		Cdd string
	}
	tests := []struct{
		name string
		typeOf reflect.Type
		expectedResult bool
	}{
		{
			name: "test type is exported and not builtin",
			typeOf: reflect.TypeOf(ExportedStruct{
				Abb: "testing",
			}),
			expectedResult: true,
		},
		{
			name: "test type is not exported and not builtin",
			typeOf: reflect.TypeOf(notExportedStruct{
				Cdd: "testing",
			}),
			expectedResult: false,
		},
		{
			name: "test type is builtin",
			typeOf: reflect.TypeOf("Abb"), // string type
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResult := isExportedOrBuiltinType(tt.typeOf)

			assert.Equal(t, tt.expectedResult, actualResult)
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
