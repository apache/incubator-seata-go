package parser

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/statelang"
	"reflect"
	"sync"
	"unicode"
)

type StateMachineParser interface {
	GetType() string
	Parse(content string) (statelang.StateMachine, error)
}

type StateParser interface {
	StateType() string
	Parse(stateName string, stateMap map[string]interface{}) (statelang.State, error)
}

type BaseStateParser struct {
}

func (b BaseStateParser) ParseBaseAttributes(stateName string, state statelang.State, stateMap map[string]interface{}) error {
	state.SetName(stateName)

	comment, err := b.GetString(stateName, stateMap, "Comment")
	if err != nil {
		return err
	}
	state.SetComment(comment)

	next, err := b.GetString(stateName, stateMap, "Next")
	if err != nil {
		return err
	}

	state.SetNext(next)
	return nil
}

func (b BaseStateParser) GetString(stateName string, stateMap map[string]interface{}, key string) (string, error) {
	value := stateMap[key]
	if value == nil {
		var result string
		return result, errors.New("State [" + stateName + "] " + key + " not exist")
	}

	valueAsString, ok := value.(string)
	if !ok {
		var s string
		return s, errors.New("State [" + stateName + "] " + key + " illegal, required string")
	}
	return valueAsString, nil
}

func (b BaseStateParser) GetSlice(stateName string, stateMap map[string]interface{}, key string) ([]interface{}, error) {
	value := stateMap[key]

	if value == nil {
		var result []interface{}
		return result, errors.New("State [" + stateName + "] " + key + " not exist")
	}

	valueAsSlice, ok := value.([]interface{})
	if !ok {
		var result []interface{}
		return result, errors.New("State [" + stateName + "] " + key + " illegal, required slice")
	}
	return valueAsSlice, nil
}

type StateParserFactory interface {
	RegistryStateParser(stateType string, stateParser StateParser)

	GetStateParser(stateType string) StateParser
}

type DefaultStateParserFactory struct {
	stateParserMap map[string]StateParser
	mutex          sync.Mutex
}

func NewDefaultStateParserFactory() *DefaultStateParserFactory {
	var stateParserMap map[string]StateParser = make(map[string]StateParser)
	return &DefaultStateParserFactory{
		stateParserMap: stateParserMap,
	}
}

// InitDefaultStateParser init StateParser by default
func (d *DefaultStateParserFactory) InitDefaultStateParser() {
	choiceStateParser := NewChoiceStateParser()

	d.RegistryStateParser(choiceStateParser.StateType(), choiceStateParser)
}

func (d *DefaultStateParserFactory) RegistryStateParser(stateType string, stateParser StateParser) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.stateParserMap[stateType] = stateParser
}

func (d *DefaultStateParserFactory) GetStateParser(stateType string) StateParser {
	return d.stateParserMap[stateType]
}

// MapToStruct some state can use this util to parse,TODO 性能测试，性能差的话，直接去解析，不使用反射
func MapToStruct(stateName string, obj interface{}, stateMap map[string]interface{}) error {
	objVal := reflect.ValueOf(obj)
	if objVal.Kind() != reflect.Pointer {
		return errors.New(fmt.Sprintf("State [%s] value required a pointer", stateName))
	}

	structValue := objVal.Elem()
	if structValue.Kind() != reflect.Struct {
		return errors.New(fmt.Sprintf("State [%s] value elem required a struct", stateName))
	}

	structType := structValue.Type()
	for key, value := range stateMap {
		//Get field, get alias first
		field, found := getField(structType, key)
		if !found {
			continue
		}

		fieldVal := structValue.FieldByName(field.Name)
		if !fieldVal.IsValid() {
			return errors.New(fmt.Sprintf("State [%s] not support [%s] filed", stateName, key))
		}

		//Get setMethod
		var setMethod reflect.Value
		if !fieldVal.CanSet() {
			setMethod = getFiledSetMethod(field.Name, objVal)

			if !setMethod.IsValid() {
				fieldAliasName := field.Tag.Get("alias")
				setMethod = getFiledSetMethod(fieldAliasName, objVal)
			}

			if !setMethod.IsValid() {
				return errors.New(fmt.Sprintf("State [%s] [%s] field not support setMethod", stateName, key))
			}
			setMethodType := setMethod.Type()
			if !(setMethodType.NumIn() == 1 && setMethodType.In(0) == fieldVal.Type()) {
				return errors.New(fmt.Sprintf("State [%s] [%s] field setMethod illegal", stateName, key))
			}
		}

		val := reflect.ValueOf(value)
		if fieldVal.Kind() == reflect.Struct {
			//map[string]interface{}
			if val.Kind() != reflect.Map {
				return errors.New(fmt.Sprintf("State [%s] [%s] field type required map", stateName, key))
			}

			err := MapToStruct(stateName, fieldVal.Addr().Interface(), value.(map[string]interface{}))
			if err != nil {
				return err
			}
		} else if fieldVal.Kind() == reflect.Slice {
			if val.Kind() != reflect.Slice {
				return errors.New(fmt.Sprintf("State [%s] [%s] field type required slice", stateName, key))
			}

			sliceType := fieldVal.Type().Elem()
			newSlice := reflect.MakeSlice(fieldVal.Type(), 0, val.Len())

			for i := 0; i < val.Len(); i++ {
				newElem := reflect.New(sliceType.Elem())
				elemMap := val.Index(i).Interface().(map[string]interface{})
				err := MapToStruct(stateName, newElem.Interface(), elemMap)
				if err != nil {
					return err
				}
				reflect.Append(newSlice, newElem.Elem())
			}
			setFiled(fieldVal, setMethod, newSlice)
		} else if fieldVal.Kind() == reflect.Map {
			if val.Kind() != reflect.Map {
				return errors.New(fmt.Sprintf("State [%s] [%s] field type required map", stateName, key))
			}

			mapType := field.Type
			newMap := reflect.MakeMap(mapType)

			for _, key := range val.MapKeys() {
				newVal := reflect.New(mapType.Elem().Elem())
				elemMap := val.MapIndex(key).Interface().(map[string]interface{})
				err := MapToStruct(stateName, newVal.Interface(), elemMap)
				if err != nil {
					return err
				}
				newMap.SetMapIndex(key, newVal.Elem())
			}
			setFiled(fieldVal, setMethod, newMap)
		} else {
			setFiled(fieldVal, setMethod, val)
		}
	}
	return nil
}

func getField(t reflect.Type, name string) (reflect.StructField, bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag, hasAliasTag := field.Tag.Lookup("alias")

		if (hasAliasTag && tag == name) || (!hasAliasTag && field.Name == name) {
			return field, true
		}

		if field.Anonymous {
			embeddedField, ok := getField(field.Type, name)
			if ok {
				return embeddedField, true
			}
		}
	}

	return reflect.StructField{}, false
}

func getFiledSetMethod(name string, structValue reflect.Value) reflect.Value {
	fieldNameSlice := []rune(name)
	fieldNameSlice[0] = unicode.ToUpper(fieldNameSlice[0])

	setMethodName := "Set" + string(fieldNameSlice)

	setMethod := structValue.MethodByName(setMethodName)
	return setMethod
}

func setFiled(fieldVal reflect.Value, setMethod reflect.Value, val reflect.Value) {
	if !fieldVal.CanSet() {
		setMethod.Call([]reflect.Value{
			val,
		})
	} else {
		fieldVal.Set(val)
	}
}
