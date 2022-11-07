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
TakePointerValue: //取指针的值
	switch typeOf.Kind() {
	case reflect.Ptr: //如果是指针类型则取地址重新判断类型
		typeOf = typeOf.Elem()
		goto TakePointerValue
	case reflect.Struct: //如果是字段结构体需要继续递归解析结构体字段所有值

	TakeValueOfPointerValue: //这里主要是考虑到有可能用的不是一级指针，如果是***int 等多级指针就需要不断的取值
		if valueOf.Kind() == reflect.Ptr {
			if valueOf.IsNil() {
				t.IsNil = true
				return
			} else {
				valueOf = valueOf.Elem()
				goto TakeValueOfPointerValue
			}
		}
		if valueOf.Convert(timeTypes).Bool() { //是time.Time类型或者底层是time.Time类型
			t.Key = key
			t.Val = valueOf.Interface()
			return
		}

		if typeOf.NumField() == 0 { //如果是一个struct{}{}类型的字段或者是一个空的自定义结构体编码为{}
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

			//是否是匿名结构体
			isAnonymous := typeOf.Field(i).Anonymous && tag.IsAnonymous //什么时候才算真正的匿名字段？ Book中Article才算匿名结构体
			//type Book struct {
			//	BookName string `json:"bookName,select(resp)"`
			//	*Page    `json:"page,select(resp)"` // 这个不算匿名字段，为什么？因为tag里打了字段名表示要当作一个字段来对待，
			//	Article    `json:",select(resp)"` //这种情况才是真正的匿名字段，因为tag里字段名为空字符串
			//}
			//

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
				if value.IsZero() { //为零值忽略
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
			//t.Val = struct{}{} //这样表示返回{}

			t.IsAnonymous = true //给他搞成匿名字段的处理方式，直接忽略字段
			//说明该结构体上没有选择任何字段 应该返回"字段名:{}"
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
		if len(keys) == 0 { //空map情况下解析为{}
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
			t.Val = nilSlice //空数组空切片直接解析为[]
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
