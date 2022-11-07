package filter

import "encoding/json"

type fieldNodeTree struct {
	Key         string           //Field name
	Val         interface{}      //The field value, basic data type, int string, bool and other types exist directly here, if it is struct, slice array map type, all k v fields will be stored in ChildNodes
	IsSlice     bool             //Whether it is a slice, or an array,
	IsAnonymous bool             //Whether it is an anonymous structure, an embedded structure, all fields need to be expanded
	IsNil       bool             //Whether the field value is nil
	ParentNode  *fieldNodeTree   //parent node pointer, the root node is nil,
	ChildNodes  []*fieldNodeTree //If it is a struct, save the pointers of all field names and values, and if it is a slice, save all the values ​​of the slice
}

func (t *fieldNodeTree) GetValue() (val interface{}, ok bool) {
	if t.IsAnonymous {
		//如果是匿名字段则不需要再追加这个字段
		return nil, false
	}
	if t.IsNil {
		return nil, true
	}
	if t.ChildNodes == nil {
		return t.Val, true
	}
	if t.IsSlice { // key is empty for slices and arrays
		slices := make([]interface{}, 0, len(t.ChildNodes))
		for i := 0; i < len(t.ChildNodes); i++ {
			value, ok0 := t.ChildNodes[i].GetValue()
			if ok0 {
				slices = append(slices, value)
			}
		}
		return slices, true
	}
	maps := make(map[string]interface{})
	for _, v := range t.ChildNodes {
		value, ok1 := v.GetValue()
		if ok1 {
			maps[v.Key] = value
		}
	}
	return maps, true
}

func (t *fieldNodeTree) Map() map[string]interface{} {
	maps := make(map[string]interface{})
	for _, v := range t.ChildNodes {
		value, ok := (*v).GetValue()
		if ok {
			maps[(*v).Key] = value
		}
	}
	return maps
}
func (t *fieldNodeTree) Slice() interface{} {
	slices := make([]interface{}, 0, len(t.ChildNodes))
	for i := 0; i < len(t.ChildNodes); i++ {
		v, ok := t.ChildNodes[i].GetValue()
		if ok {
			slices = append(slices, v)
		}
	}
	return slices
}

func (t *fieldNodeTree) Marshal() interface{} {
	if t.IsSlice {
		return t.Slice()
	} else { //说明是结构体或者map
		return t.Map()
	}
}

func (t *fieldNodeTree) AddChild(tree *fieldNodeTree) *fieldNodeTree {
	if t.ChildNodes == nil {
		t.ChildNodes = make([]*fieldNodeTree, 0, 3)
	}
	t.ChildNodes = append(t.ChildNodes, tree)
	return t
}

// GetParentNodeInsertPosition recursively finds the topmost insertable node
func (t *fieldNodeTree) GetParentNodeInsertPosition() *fieldNodeTree {
	if t.ParentNode == nil {
		return t
	}

	//Recursively to the parent node layer by layer, until a node that is not an anonymous field is found, and data is added to the child of the node
	if t.ParentNode.IsAnonymous {
		return t.ParentNode.GetParentNodeInsertPosition()
	}
	return t.ParentNode
}

// AnonymousAddChild Anonymous field append operation to the parent node
func (t *fieldNodeTree) AnonymousAddChild(tree *fieldNodeTree) *fieldNodeTree {
	t.GetParentNodeInsertPosition().AddChild(tree)
	return t
}

// MustJSON will panic directly if parsing fails
func (t *fieldNodeTree) MustJSON() string {
	j, err := json.Marshal(t.Marshal())
	//j, err := sonic.Marshal(t.Marshal()) //这个目前兼容性不是特别好，先用官方库
	if err != nil {
		panic(err)
	}
	return string(j)
}

func (t *fieldNodeTree) JSON() (string, error) {
	j, err := json.Marshal(t.Marshal())
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func (t *fieldNodeTree) Bytes() ([]byte, error) {
	j, err := json.Marshal(t.Marshal())
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (t *fieldNodeTree) MustBytes() []byte {
	j, err := json.Marshal(t.Marshal())
	if err != nil {
		panic(err)
	}
	return j
}
