package remoting

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/tm"
	"reflect"
	"testing"
)

var (
	typError           = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()
	typContext         = reflect.Zero(reflect.TypeOf((*context.Context)(nil)).Elem()).Type()
	typBusinessContext = reflect.Zero(reflect.TypeOf((*tm.BusinessActionContext)(nil)).Elem()).Type()
)

type TCCService struct {
	Prepare  func(ctx context.Context, params interface{}) error                             `seataTwoPhaseAction:"prepare", seataTwoPhaseName:"BusinessTCCService"`
	Commit   func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"commit"`
	Rollback func(ctx context.Context, businessActionContext tm.BusinessActionContext) error `seataTwoPhaseAction:"rollback"`
}

func TestParseRemoting(test *testing.T) {
	Parse(&TCCService{})

}

func Parse(v interface{}) error {
	valueOf := reflect.ValueOf(v)
	valueOfElem := valueOf.Elem()

	var m1, m2, m3 string

	typeOf := valueOfElem.Type()
	// check incoming interface, incoming interface's elem must be a struct.
	if typeOf.Kind() != reflect.Struct {
		return errors.New("invalid type kind")
	}

	numField := valueOfElem.NumField()
	for i := 0; i < numField; i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)
		if m, ok := validPrepareMethod(t, f); ok {
			m1 = m
			fmt.Sprintln("prepare")
		} else if m, ok = validCommitMethod(t, f); ok {
			m2 = m
			fmt.Sprintln("commit")
		} else if m, ok = validRollbackMethod(t, f); ok {
			m3 = m
			fmt.Sprintln("rollback")
		}
	}
	fmt.Sprintln(m1, m2, m3)
	return nil
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
		if actionName = t.Tag.Get("seataTwoPhaseName"); actionName != "" {
			break
		}
	}
	return actionName
}

func validPrepareMethod(t reflect.StructField, f reflect.Value) (string, bool) {
	if t.Tag.Get("seataTwoPhase") != "prepare" {
		return "", false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", false
	}
	// prepare has 1 retuen error value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", false
	}
	// prepared method has 2 params, context.Context, interface{}
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", false
	}
	return t.Name, true
}

func validCommitMethod(t reflect.StructField, f reflect.Value) (string, bool) {
	if t.Tag.Get("seataTwoPhase") != "commit" {
		return "", false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", false
	}
	// commit method has 1 retuen error value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", false
	}
	// commit method has 2 params, context.Context, tm.BusinessActionContext
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", false
	}
	if inType := t.Type.In(1); inType != typBusinessContext {
		return "", false
	}
	return t.Name, true

}

func validRollbackMethod(t reflect.StructField, f reflect.Value) (string, bool) {
	if t.Tag.Get("seataTwoPhase") != "rollback" {
		return "", false
	}
	if f.Kind() != reflect.Func || !f.IsValid() {
		return "", false
	}
	// rollback method has 1 retuen value
	if outNum := t.Type.NumOut(); outNum != 1 {
		return "", false
	}
	if returnType := t.Type.Out(0); returnType != typError {
		return "", false
	}
	// rollback method has 2 params, context.Context, tm.BusinessActionContext
	if inNum := t.Type.NumIn(); inNum != 2 {
		return "", false
	}
	if inType := t.Type.In(0); inType != typContext {
		return "", false
	}
	if inType := t.Type.In(1); inType != typBusinessContext {
		return "", false
	}
	return t.Name, true
}
