/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package reflectx

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"unicode"
)

// MapToStruct some state can use this util to parse
// TODO 性能测试，性能差的话，直接去解析，不使用反射
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
