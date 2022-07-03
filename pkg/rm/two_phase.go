package rm

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/tm"
	"reflect"
)

const (
	TwoPhaseActionTag            = "seataTwoPhaseAction"
	TwoPhaseActionNameTag        = "seataTwoPhaseServiceName"
	TwoPhaseActionPrepareTagVal  = "prepare"
	TwoPhaseActionCommitTagVal   = "commit"
	TwoPhaseActionRollbackTagVal = "rollback"
)

var (
	typError           = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
	typContext         = reflect.Zero(reflect.TypeOf((*context.Context)(nil)).Elem()).Type()
	typBusinessContext = reflect.Zero(reflect.TypeOf((*tm.BusinessActionContext)(nil)).Elem()).Type()
)

type TwoPhaseInterface interface {
	Prepare(ctx context.Context, params ...interface{}) error
	Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error
	Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error
	GetActionName() string
}

type TwoPhaseAction struct {
	twoPhaseService    interface{}
	actionName         string
	prepareMethodName  string
	prepareMethod      *reflect.Value
	commitMethodName   string
	commitMethod       *reflect.Value
	rollbackMethodName string
	rollbackMethod     *reflect.Value
}

func (t *TwoPhaseAction) Prepare(ctx context.Context, params ...interface{}) error {
	res := t.prepareMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(params)})
	returnVal := res[0]
	return returnVal.Interface().(error)
}

func (t *TwoPhaseAction) Commit(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	res := t.commitMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(businessActionContext)})
	returnVal := res[0]
	return returnVal.Interface().(error)
}

func (t *TwoPhaseAction) Rollback(ctx context.Context, businessActionContext tm.BusinessActionContext) error {
	res := t.rollbackMethod.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(businessActionContext)})
	returnVal := res[0]
	return returnVal.Interface().(error)
}

func (t *TwoPhaseAction) GetActionName() string {
	return t.actionName
}

func IsTwoPhaseAction(v interface{}) bool {
	m, err := ParseTwoPhaseAction(v)
	return m != nil && err != nil
}

func ParseTwoPhaseAction(v interface{}) (*TwoPhaseAction, error) {
	if m, ok := v.(TwoPhaseInterface); ok {
		return parseTwoPhaseActionByTwoPhaseInterface(m), nil
	}
	return ParseTwoPhaseActionByInterface(v)
}

func parseTwoPhaseActionByTwoPhaseInterface(v TwoPhaseInterface) *TwoPhaseAction {
	value := reflect.ValueOf(v)
	mp := value.MethodByName("Prepare")
	mc := value.MethodByName("Commit")
	mr := value.MethodByName("Rollback")
	return &TwoPhaseAction{
		twoPhaseService:    v,
		actionName:         v.GetActionName(),
		prepareMethodName:  "Prepare",
		prepareMethod:      &mp,
		commitMethodName:   "Commit",
		commitMethod:       &mc,
		rollbackMethodName: "Rollback",
		rollbackMethod:     &mr,
	}
}

func ParseTwoPhaseActionByInterface(v interface{}) (*TwoPhaseAction, error) {
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
		result               = TwoPhaseAction{
			twoPhaseService: v,
		}
	)
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if ms, m, ok := getPrepareAction(t, f); ok {
			hasPrepareMethodName = true
			result.prepareMethod = m
			result.prepareMethodName = ms
		} else if ms, m, ok = getCommitMethod(t, f); ok {
			hasCommitMethodName = true
			result.commitMethod = m
			result.commitMethodName = ms
		} else if ms, m, ok = getRollbackMethod(t, f); ok {
			hasRollbackMethod = true
			result.rollbackMethod = m
			result.rollbackMethodName = ms
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
	result.actionName = twoPhaseName
	return &result, nil
}

func getPrepareAction(t reflect.StructField, f reflect.Value) (string, *reflect.Value, bool) {
	if t.Tag.Get(TwoPhaseActionTag) != TwoPhaseActionPrepareTagVal {
		return "", nil, false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", nil, false
	}
	// prepare has 1 retuen error value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", nil, false
	}
	// prepared method has 2 params, context.Context, interface{}
	if inNum := t.Type.NumIn(); inNum != 2 {
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
	// commit method has 1 retuen error value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", nil, false
	}
	// commit method has 2 params, context.Context, tm.BusinessActionContext
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	if inType := t.Type.In(1); inType != typBusinessContext {
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
	// rollback method has 1 retuen value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", nil, false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", nil, false
	}
	// rollback method has 2 params, context.Context, tm.BusinessActionContext
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", nil, false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", nil, false
	}
	if inType := t.Type.In(1); inType != typBusinessContext {
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
