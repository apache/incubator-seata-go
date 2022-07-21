package rm

import (
	"reflect"
	"regexp"

	"github.com/seata/seata-go/pkg/common/log"
)

type grpcRemotingParse struct {
}

func (parse *grpcRemotingParse) ParseService(target interface{}) (*TwoPhaseAction, error) {
	//TODO implement me
	panic("implement me")
}

func (parse *grpcRemotingParse) ParseReference(target interface{}) (*TwoPhaseAction, error) {
	//TODO implement me
	panic("implement me")
}

func (parse *grpcRemotingParse) ParseTwoPhaseActionByInterface(target interface{}) (*TwoPhaseAction, error) {
	//TODO implement me
	panic("implement me")
}

func (parse *grpcRemotingParse) IsRemoting(target interface{}) bool {
	return parse.IsService(target) || parse.IsReference(target)
}

func (parse *grpcRemotingParse) IsService(target interface{}) bool {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	fieldTag := v.FieldByNameFunc(parse.matchServiceTag)
	if fieldTag.IsNil() {
		log.Errorf("target %v is not grpc service", target)
		return false
	}
	fieldType := fieldTag.Type()
	lastMethod := fieldType.Method(fieldType.NumMethod() - 1)
	if ok, err := regexp.MatchString(`\\`, lastMethod.Name); err != nil {
		log.Errorf("regex failed in match grpc service method name ")
		return false
	} else {
		return ok
	}
}

func (parse *grpcRemotingParse) IsReference(target interface{}) bool {
	v := reflect.ValueOf(target)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	cc := v.FieldByName("cc")
	if cc.Type().String() == "ClientConnInterface" {
		return true
	} else {
		return false
	}
}

func (parse *grpcRemotingParse) matchServiceTag(fieldName string) bool {
	if ok, err := regexp.MatchString(``, fieldName); err != nil {
		log.Errorf("regex failed in match grpc service method name ")
		return false
	} else {
		return ok
	}
}

func (parse grpcRemotingParse) GetRemotingType(target interface{}) int {
	if parse.IsReference(target) || parse.IsService(target) {
		return RemotingGrpc
	} else {
		return RemotingUnknow
	}
}

func init() {
	RegisterRemotingParse(&grpcRemotingParse{})
}
