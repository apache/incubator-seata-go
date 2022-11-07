package filter

import (
	"reflect"
)

func (t *fieldNodeTree) ParseOmitValue(key, omitScene string, el interface{}) {

	typeOf := reflect.TypeOf(el)
	valueOf := reflect.ValueOf(el)
TakePointerValue: //get pointer value
	switch typeOf.Kind() {
	case reflect.Ptr: //If it is a pointer type, take the address and re-judg the type
		typeOf = typeOf.Elem()
		goto TakePointerValue
	case reflect.Struct: //If it is a field structure, you need to continue to recursively parse all the values of the structure field

	TakeValueOfPointerValue: //The main reason here is to consider that it may not be a first-level pointer. If it is a multi-level pointer such as ***int, it needs to continuously take values.
		if valueOf.Kind() == reflect.Ptr {
			if valueOf.IsNil() {
				t.IsNil = true
				//tree.IsNil=true
				//t.AddChild(tree)
				return
			} else {
				valueOf = valueOf.Elem()
				goto TakeValueOfPointerValue
			}
		}

		if valueOf.Convert(timeTypes).Bool() {
			t.Key = key
			t.Val = valueOf.Interface()
			return
		}
		if typeOf.NumField() == 0 { //If it is a field of type struct{}{} or an empty custom structure encoded as {}
			t.Key = key
			t.Val = struct{}{}
			return
		}
		var isAnonymous bool
		for i := 0; i < typeOf.NumField(); i++ {
			jsonTag, ok := typeOf.Field(i).Tag.Lookup("json")
			var tag Tag
			if !ok {
				tag = newOmitNotTag(omitScene, typeOf.Field(i).Name)
				isAnonymous = typeOf.Field(i).Anonymous
			} else {
				if jsonTag == "-" {
					continue
				}
				tag = newOmitTag(jsonTag, omitScene, typeOf.Field(i).Name)
				if tag.IsOmitField || !tag.IsSelect {
					continue
				}
				isAnonymous = typeOf.Field(i).Anonymous && tag.IsAnonymous
			}

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

			tree.ParseOmitValue(tag.UseFieldName, omitScene, value.Interface())

			if t.IsAnonymous {
				t.AnonymousAddChild(tree)
			} else {
				t.AddChild(tree)
			}
		}
		if t.ChildNodes == nil && !t.IsAnonymous {

			t.IsAnonymous = true
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
				nodeTree.ParseOmitValue(k, omitScene, val.Interface())
				t.AddChild(nodeTree)
			}
		}

	case reflect.Slice, reflect.Array:
		l := valueOf.Len()
		if l == 0 {
			t.Val = nilSlice // Avoid empty arrays from resolving to null
			return
		}
		t.IsSlice = true
		for i := 0; i < l; i++ {
			sliceIsNil := false

			//node := newFieldNodeTree("", t)
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
				node.ParseOmitValue("", omitScene, valueOf.Index(i).Interface())
				t.AddChild(node)
			}
		}
	}
}
