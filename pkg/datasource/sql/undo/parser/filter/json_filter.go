package filter

type Filter struct {
	node *fieldNodeTree
}

func (f Filter) MarshalJSON() ([]byte, error) {
	return f.node.Bytes()
}

//Deprecated
func (f Filter) MastMarshalJSON() []byte {
	return f.node.MustBytes()
}
func (f Filter) MustMarshalJSON() []byte {
	return f.node.MustBytes()
}

// Interface resolves to map[string]interface{} to be json serialized after filtering
func (f Filter) Interface() interface{} {
	return f.node.Marshal()
}

// MustJSON gets the parsed and filtered json string, if there is an error in the middle, it will panic
func (f Filter) MustJSON() string {
	return f.node.MustJSON()
}

// JSON Get the parsed and filtered json string, if there is an error in the middle, it will return an error
func (f Filter) JSON() (string, error) {
	return f.node.JSON()
}

// String fmt.Println() output json string when printing
func (f Filter) String() string {
	json, err := f.JSON()
	if err != nil {
		return "[Filter Err]"
	}
	return json
}

// The first parameter of SelectMarshal fills in the scene in the select tag of your structure, and the second parameter is the structure object you need to filter. If the scene is marked in the select tag of the field, the field will be selected.
func SelectMarshal(selectScene string, el interface{}) Filter {
	tree := &fieldNodeTree{
		Key:        "",
		ParentNode: nil,
	}
	tree.ParseSelectValue("", selectScene, el)
	return Filter{
		node: tree,
	}
}

// The first parameter of OmitMarshal fills in the scene in the omit tag of your structure, and the second parameter is the structure object you need to filter. If the scene is marked in the omit tag of the field, the field will be filtered out
func OmitMarshal(omitScene string, el interface{}) Filter {
	tree := &fieldNodeTree{
		Key:        "",
		ParentNode: nil,
	}
	tree.ParseOmitValue("", omitScene, el)
	return Filter{
		node: tree,
	}
}
