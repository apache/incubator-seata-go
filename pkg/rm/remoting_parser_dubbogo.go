package rm

import (
	"context"
	"reflect"

	"github.com/seata/seata-go/pkg/tm"

	"github.com/pkg/errors"
)

var (
	typError                    = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
	typContext                  = reflect.Zero(reflect.TypeOf((*context.Context)(nil)).Elem()).Type()
	typBool                     = reflect.Zero(reflect.TypeOf((*bool)(nil)).Elem()).Type()
	typBusinessContextInterface = reflect.Zero(reflect.TypeOf((*tm.BusinessActionContext)(nil))).Type()
)

func init() {
	RegisterRemotingParse(&dubbogoRemotingParse{})
}

type dubbogoRemotingParse struct {
}

func (parse *dubbogoRemotingParse) ParseService(v interface{}) (*TwoPhaseAction, error) {
	valueOfElem := reflect.ValueOf(v).Elem()
	typeOf := valueOfElem.Type()
	k := typeOf.Kind()
	if k != reflect.Struct {
		return nil, errors.New("invalid type kind")
	}
	numField := typeOf.NumField()
	if typeOf.Kind() != reflect.Struct {
		return nil, errors.New("param should be a struct, instead of a pointer")
	}

	var (
		hasPrepareMethodName bool
		hasCommitMethodName  bool
		hasRollbackMethod    bool
		twoPhaseName         string
		result               = TwoPhaseAction{}
	)
	result.SetTwoPhaseService(v)
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if ms, m, ok := getPrepareAction(t, f); ok {
			hasPrepareMethodName = true
			result.SetPrepareMethod(m)
			result.SetPrepareMethodName(ms)
		} else if ms, m, ok = getCommitMethod(t, f); ok {
			hasCommitMethodName = true
			result.SetCommitMethod(m)
			result.SetCommitMethodName(ms)
		} else if ms, m, ok = getRollbackMethod(t, f); ok {
			hasRollbackMethod = true
			result.SetRollbackMethod(m)
			result.SetRollbackMethodName(ms)
		}
	}
	if !hasPrepareMethodName {
		return nil, errors.New("missing prepare method")
	}
	if !hasCommitMethodName {
		return nil, errors.New("missing commit method")
	}
	if !hasRollbackMethod {
		return nil, errors.New("missing rollback method")
	}
	twoPhaseName = getActionName(v)
	if twoPhaseName == "" {
		return nil, errors.New("missing two phase name")
	}
	result.SetActionName(twoPhaseName)
	return &result, nil
}
func (parse *dubbogoRemotingParse) ParseReference(v interface{}) (*TwoPhaseAction, error) {
	valueOfElem := reflect.ValueOf(v).Elem()
	typeOf := valueOfElem.Type()
	k := typeOf.Kind()
	if k != reflect.Struct {
		return nil, errors.New("invalid type kind")
	}
	numField := typeOf.NumField()
	if typeOf.Kind() != reflect.Struct {
		return nil, errors.New("param should be a struct, instead of a pointer")
	}

	var (
		hasPrepareMethodName bool
		hasCommitMethodName  bool
		hasRollbackMethod    bool
		twoPhaseName         string
		result               = TwoPhaseAction{}
	)
	result.SetTwoPhaseService(v)
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if ms, m, ok := getPrepareAction(t, f); ok {
			hasPrepareMethodName = true
			result.SetPrepareMethod(m)
			result.SetPrepareMethodName(ms)
		} else if ms, m, ok = getCommitMethod(t, f); ok {
			hasCommitMethodName = true
			result.SetCommitMethod(m)
			result.SetCommitMethodName(ms)
		} else if ms, m, ok = getRollbackMethod(t, f); ok {
			hasRollbackMethod = true
			result.SetRollbackMethod(m)
			result.SetRollbackMethodName(ms)
		}
	}
	if !hasPrepareMethodName {
		return nil, errors.New("missing prepare method")
	}
	if !hasCommitMethodName {
		return nil, errors.New("missing commit method")
	}
	if !hasRollbackMethod {
		return nil, errors.New("missing rollback method")
	}
	twoPhaseName = getActionName(v)
	if twoPhaseName == "" {
		return nil, errors.New("missing two phase name")
	}
	result.SetActionName(twoPhaseName)
	return &result, nil
}

func (parse *dubbogoRemotingParse) IsRemoting(target interface{}) bool {
	return parse.IsReference(target) || parse.IsService(target)
}

func (parse *dubbogoRemotingParse) IsService(target interface{}) bool {
	t := reflect.ValueOf(target)
	if t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	methodNum := t.NumMethod()
	fieldNum := t.Elem().NumField()
	return methodNum == 4 && fieldNum == 0
}

func (parse *dubbogoRemotingParse) IsReference(target interface{}) bool {
	t := reflect.ValueOf(target)
	if t.Kind() == reflect.Interface {
		t = t.Elem()
	}
	methodNum := t.NumMethod()
	fieldNum := t.Elem().NumField()
	return methodNum == 0 && fieldNum == 4
}

func (parse dubbogoRemotingParse) GetRemotingType(target interface{}) int {
	if parse.IsRemoting(target) {
		return RemotingDubbogo
	} else {
		return RemotingUnknow
	}
}

func (parse *dubbogoRemotingParse) ParseTwoPhaseActionByInterface(v interface{}) (*TwoPhaseAction, error) {
	if !parse.IsRemoting(v) {
		return nil, nil
	} else {
		if parse.IsReference(v) {
			return parse.ParseReference(v)
		} else if parse.IsService(v) {
			return parse.ParseService(v)
		}
	}
	return nil, errors.New("unexpected action occur in dubbogoRemotingParse#ParseTwoPhaseActionByInterface ")
}

func getPrepareAction(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionPrepareTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// prepare has 2 retuen error value， any type and error.
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	// prepare has 2 return param, one is interface{}, another is error.
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// prepared method has at least 1 params, context.Context, and other params
	if inNum := t.Type.NumIn(); inNum == 0 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	return t.Name, &f, true
}

func getCommitMethod(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionCommitTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// commit method has 2 retuen error value
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typBool {
		return "", nil, false
	}
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// commit method has at least 1 params, context.Context, and other params
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	if inType := t.Type.In(1); inType != typBusinessContextInterface {
		return "", nil, false
	}
	return t.Name, &f, true

}

func getRollbackMethod(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionRollbackTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// rollback method has 2 retuen value
	if outNum := t.Type.NumOut(); outNum != 2 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typBool {
		return "", nil, false
	}
	if returnType := t.Type.Out(1); returnType != typError {
		return "", nil, false
	}
	// rollback method has at least 1 params, context.Context, and other params
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	if inType := t.Type.In(1); inType != typBusinessContextInterface {
		return "", nil, false
	}
	return t.Name, &f, true
}

func getActionName(v interface{}) string {
	var (
		actionName  string
		valueOf     = reflect.ValueOf(v)
		valueOfElem = valueOf.Elem()
		typeOf      = valueOfElem.Type()
	)
	if typeOf.Kind() != reflect.Struct {
		return ""
	}
	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		if actionName = t.Tag.Get(TwoPhaseActionNameTag); actionName != "" {
			break
		}
	}
	return actionName
}
