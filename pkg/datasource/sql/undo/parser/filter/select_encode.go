package filter

import (
	"reflect"
	"time"
)

var (
	nilSlice  = make([]int, 0, 0)
	timeTypes = reflect.TypeOf(time.Now())
)

// ParseSelectValue 解析字段值
func (t *fieldNodeTree) ParseSelectValue(key, selectScene string, el interface{}) {

	typeOf := reflect.TypeOf(el)
	valueOf := reflect.ValueOf(el)
TakePointerValue:
	switch typeOf.Kind() {
	case reflect.Ptr: //If it is a pointer type, take the address and re-judg the type
		typeOf = typeOf.Elem()
		goto TakePointerValue
	case reflect.Struct: //If it is a field structure, you need to continue to recursively parse all the values of the structure field

	TakeValueOfPointerValue:
		if valueOf.Kind() == reflect.Ptr {
			if valueOf.IsNil() {
				t.IsNil = true
				return
			} else {
				valueOf = valueOf.Elem()
				goto TakeValueOfPointerValue
			}
		}
		if valueOf.Convert(timeTypes).Bool() { //is the time.Time type or the bottom layer is the time.Time type
			t.Key = key
			t.Val = valueOf.Interface()
			return
		}

		if typeOf.NumField() == 0 { //If it is a field of type struct{}{} or an empty custom structure encoded as {}
			t.Key = key
			t.Val = struct{}{}
			return
		}

		for i := 0; i < typeOf.NumField(); i++ {
			jsonTag, ok := typeOf.Field(i).Tag.Lookup("json")
			if !ok || jsonTag == "-" {
				continue
			}
			tag := newSelectTag(jsonTag, selectScene, typeOf.Field(i).Name)
			if tag.IsOmitField || !tag.IsSelect {
				continue
			}

			// Is it an anonymous structure
			isAnonymous := typeOf.Field(i).Anonymous && tag.IsAnonymous // anonymous field

			tree := &fieldNodeTree{
				Key:         tag.UseFieldName,
				ParentNode:  t,
				IsAnonymous: isAnonymous,
			}

			value := valueOf.Field(i)
		TakeFieldValue:
			if value.Kind() == reflect.Ptr {
				if value.IsNil() {
					if tag.Omitempty {
						continue
					}
					tree.IsNil = true
					t.AddChild(tree)
					continue
				} else {
					value = value.Elem()
					goto TakeFieldValue
				}
			}

			if tag.Omitempty {
				if value.IsZero() {
					continue
				}
			}
			tree.ParseSelectValue(tag.UseFieldName, selectScene, value.Interface())

			if t.IsAnonymous {
				t.AnonymousAddChild(tree)
			} else {
				t.AddChild(tree)
			}
		}
		if t.ChildNodes == nil && !t.IsAnonymous {

			t.IsAnonymous = true // ignore fields Indicates that no field is selected on the structure, should return "field name: {}"
		}
	case reflect.Bool,
		reflect.String,
		reflect.Float64, reflect.Float32,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:

		if t.IsAnonymous {
			tree := &fieldNodeTree{
				Key:        t.Key,
				ParentNode: t,
				Val:        t.Val,
			}
			t.AnonymousAddChild(tree)
		} else {
			t.Val = valueOf.Interface()
			t.Key = key
		}

	case reflect.Map:
		if valueOf.Kind() == reflect.Ptr {
			valueOf = valueOf.Elem()
		}
		keys := valueOf.MapKeys()
		if len(keys) == 0 {
			t.Val = struct{}{}
			return
		}
		for i := 0; i < len(keys); i++ {
			mapIsNil := false
		takeValMap:
			val := valueOf.MapIndex(keys[i])
			if val.Kind() == reflect.Ptr {
				if val.IsNil() {
					mapIsNil = true
					continue
				} else {
					val = valueOf.MapIndex(keys[i]).Elem()
					goto takeValMap
				}
			}
			k := keys[i].String()
			nodeTree := &fieldNodeTree{
				Key:        k,
				ParentNode: t,
			}
			if mapIsNil {
				nodeTree.IsNil = true
				t.AddChild(nodeTree)
			} else {
				nodeTree.ParseSelectValue(k, selectScene, val.Interface())
				t.AddChild(nodeTree)
			}
		}

	case reflect.Slice, reflect.Array:
		l := valueOf.Len()
		if l == 0 {
			t.Val = nilSlice
			return
		}
		t.IsSlice = true
		for i := 0; i < l; i++ {
			sliceIsNil := false
			node := &fieldNodeTree{
				Key:        "",
				ParentNode: t,
			}
			val := valueOf.Index(i)
		takeValSlice:
			if val.Kind() == reflect.Ptr {
				if val.IsNil() {
					sliceIsNil = true
					continue
				} else {
					val = val.Elem()
					goto takeValSlice
				}
			}
			if sliceIsNil {
				node.IsNil = true
				t.AddChild(node)
			} else {
				node.ParseSelectValue("", selectScene, valueOf.Index(i).Interface())
				t.AddChild(node)
			}
		}
	}
}
