package filter

import (
	"fmt"
	"testing"
	"time"
)

type MyTime = time.Time
type YouTime = MyTime

type TestCases struct {
	Article `json:"article,select(all|Anonymous)"`

	*Anonymous `json:",select(all|Anonymous)"`

	Time   time.Time  `json:"time,select(all)"`
	TimeP  *time.Time `json:"time_p,select(all)"`
	MTime  MyTime     `json:"m_time,select(all)"`
	MTimeP *MyTime    `json:"m_time_p,select(all)"`
	YTime  YouTime    `json:"y_time,select(all)"`
	YTimeP *YouTime   `json:"y_time_p,select(all)"`

	Int    int    `json:"int,select(all|intAll)"`
	Int8   int8   `json:"int8,select(all|intAll)"`
	Int16  int16  `json:"int16,select(all|intAll)"`
	Int32  int32  `json:"int32,select(all|intAll)"`
	Int64  int64  `json:"int64,select(all|intAll)"`
	IntP   *int   `json:"int_p,select(all|intAll)"`
	Int8P  *int8  `json:"int8_p,select(all|intAll)"`
	Int16P *int16 `json:"int16_p,select(all|intAll)"`
	Int32P *int32 `json:"int32_p,select(all|intAll)"`
	Int64P *int64 `json:"int64_p,select(all|intAll)"`

	UInt    uint    `json:"u_int,select(all)"`
	UInt8   uint8   `json:"u_int8,select(all)"`
	UInt16  uint16  `json:"u_int16,select(all)"`
	UInt32  uint32  `json:"u_int32,select(all)"`
	UInt64  uint64  `json:"u_int64,select(all)"`
	UIntP   *uint   `json:"u_intP,select(all)"`
	UInt8P  *uint8  `json:"u_int_8_p,select(all)"`
	UInt16P *uint16 `json:"u_int_16_p,select(all)"`
	UInt32P *uint32 `json:"u_int_32_p,select(all)"`
	UInt64P *uint64 `json:"u_int_64_p,select(all)"`

	Float64  float64  `json:"float64,select(all)"`
	Float64P *float64 `json:"float_64_p,select(all)"`
	Float32  float32  `json:"float32,select(all)"`
	Float32P *float32 `json:"float_32_p,select(all)"`

	Bool  bool  `json:"bool,select(all)"`
	BoolP *bool `json:"bool_p,select(all)"`

	Byte  byte  `json:"byte,select(all|byteAll)"`
	ByteP *byte `json:"byte_p,select(all|byteAll)"`

	String  string  `json:"string,select(all)"`
	StringP *string `json:"string_p,select(all)"`

	Interface  interface{} `json:"interface,select(all)"`
	InterfaceP interface{} `json:"interface_p,select(all)"`

	Struct struct{} `json:"struct,select(all|struct)"`

	StructEl struct {
		Name string `json:"name,select(all|struct)"`
	} `json:"struct_el,select(all|struct)"`

	StructP *struct{} `json:"struct_p,select(all|struct)"`

	Structs  UsersCase `json:"structs,select(all|struct)"`
	StructsP *UserP    `json:"structs_p,select(all|struct)"`

	Map  map[string]interface{}  `json:"map,select(all|mapAll)"`
	MapP *map[string]interface{} `json:"map_p,select(all|mapAll)"`

	SliceInt        []int         `json:"slice_int,select(all|sliceAll)"`
	SliceInt8       []int8        `json:"slice_int_8,select(all|sliceAll)"`
	SliceInt16      []int16       `json:"slice_int_16,select(all|sliceAll)"`
	SliceInt32      []int32       `json:"slice_int_32,select(all|sliceAll)"`
	SliceInt64      []int64       `json:"slice_int_64,select(all|sliceAll)"`
	SliceIntP       []*int        `json:"slice_int_p,select(all|sliceAll)"`
	SliceInt8P      []*int8       `json:"slice_int_8_p,select(all|sliceAll)"`
	SliceInt16P     []*int16      `json:"slice_int_16_p,select(all|sliceAll)"`
	SliceInt32I     []*int32      `json:"slice_int_32_i,select(all|sliceAll)"`
	SliceInt64I     []*int64      `json:"slice_int_64_i,select(all|sliceAll)"`
	SliceUint       []uint        `json:"slice_uint,select(all|sliceAll)"`
	SliceUint8      []uint8       `json:"slice_uint_8,select(all|sliceAll)"`
	SliceUint16     []uint16      `json:"slice_uint_16,select(all|sliceAll)"`
	SliceUint32     []uint32      `json:"slice_uint_32,select(all|sliceAll)"`
	SliceUint64     []uint64      `json:"slice_uint_64,select(all|sliceAll)"`
	SliceUintP      []*uint       `json:"slice_uint_p,select(all|sliceAll)"`
	SliceUint8P     []*uint8      `json:"slice_uint_8_p,select(all|sliceAll)"`
	SliceUint16P    []*uint16     `json:"slice_uint_16_p,select(all|sliceAll)"`
	SliceUint32P    []*uint32     `json:"slice_uint_32_p,select(all|sliceAll)"`
	SliceUint64P    []*uint64     `json:"slice_uint_64_p,select(all|sliceAll)"`
	SliceFloat64    []float64     `json:"slice_float_64,select(all|sliceAll)"`
	SliceFloat64P   []*float64    `json:"slice_float_64_p,select(all|sliceAll)"`
	SliceFloat32    []float32     `json:"slice_float_32,select(all|sliceAll)"`
	SliceFloat32P   []*float32    `json:"slice_float_32_p,select(all|sliceAll)"`
	SliceBool       []bool        `json:"slice_bool,select(all|sliceAll)"`
	SliceBoolP      []*bool       `json:"slice_bool_p,select(all|sliceAll)"`
	SliceByte       []byte        `json:"slice_byte,select(all|sliceAll)"`
	SliceByteP      []*byte       `json:"slice_byte_p,select(all|sliceAll)"`
	SliceString     []string      `json:"slice_string,select(all|sliceAll)"`
	SliceStringS    []*string     `json:"slice_string_s,select(all|sliceAll)"`
	SliceInterface  []interface{} `json:"slice_interface,select(all|sliceAll)"`
	SliceInterfaceP []interface{} `json:"slice_interface_p,select(all|sliceAll)"`
	SliceStruct     []struct{}    `json:"slice_struct,select(all|sliceAll)"`
	SliceStructP    []*struct{}   `json:"slice_struct_p,select(all|sliceAll)"`

	SliceTime      []time.Time               `json:"slice_time,select(all|sliceAll)"`
	SliceUsersCase []UsersCase               `json:"slice_users_case,select(all|sliceAll)"`
	SliceUserP     []*UserP                  `json:"slice_user_p,select(all|sliceAll)"`
	SliceMap       []map[string]interface{}  `json:"slice_map,select(all|sliceAll)"`
	SliceMapP      []*map[string]interface{} `json:"slice_map_p,select(all|sliceAll)"`

	SliceSliceInt       [][]int       `json:"slice_slice_int,select(all|sliceAll)"`
	SliceSliceUsersCase [][]UsersCase `json:"slice_slice_users_case,select(all|sliceAll)"`
}

type Child struct {
	CName string `json:"c_name,select(all|2|struct)"`
	CAge  int    `json:"c_age,select(all|struct)"`
}
type UsersCase struct {
	Name   string `json:"name,select(all|1)"`
	Age    int    `json:"age,select(all|2)"`
	Struct Child  `json:"struct,select(all|struct)"`
}

type ChildP struct {
	CName *string `json:"c_name,select(all|2)"`
	CAge  *int    `json:"c_age,select(all|struct)"`
}
type UserP struct {
	Name   *string `json:"name,select(all|1)"`
	Age    *int    `json:"age,select(all|2)"`
	Struct *ChildP `json:"struct,select(all|struct)"`
}

type BaseInfo struct {
	Name     string `json:"name,select(all|Anonymous)"`
	Title    string `json:"title,select(all|Anonymous)"`
	PageInfo `json:",select(all|Anonymous)"`
}

type PageInfo struct {
	PageNum  int `json:"page_num,select(all|Anonymous)"`
	PageSize int `json:"page_size,select(all|Anonymous)"`
}

type Article struct {
	Price    string `json:"price,select(all|Anonymous)"`
	BaseInfo `json:",select(all|Anonymous)"`
}

type AnonymousValue struct {
	AnonymousValueName string `json:"anonymous_value_name,select(all|Anonymous)"`
}
type Anonymous struct {
	AnonymousValue `json:",select(all|Anonymous)"`
}

func NewTestCases() TestCases {

	Int := 100
	Int8 := int8(8)
	Int16 := int16(16)
	Int32 := int32(32)
	Int64 := int64(64)

	uInt := uint(100)
	uInt8 := uint8(8)
	uInt16 := uint16(16)
	uInt32 := uint32(32)
	uInt64 := uint64(64)

	Bool := true
	f32p := float32(320.1)
	f64p := 320.1
	Byte := byte(10)
	nameP := "nameP"
	str := "string p"
	interfaces := "interface p"
	ageP := 10
	tests := TestCases{
		Anonymous: &Anonymous{
			AnonymousValue{
				AnonymousValueName: "anonymous_value_name",
			},
		},
		Int:      100,
		Int8:     8,
		Int16:    16,
		Int32:    32,
		Int64:    64,
		IntP:     &Int,
		Int8P:    &Int8,
		Int16P:   &Int16,
		Int32P:   &Int32,
		Int64P:   &Int64,
		UInt:     uint(1000),
		UInt8:    uint8(80),
		UInt16:   uint16(160),
		UInt32:   uint32(320),
		UInt64:   uint64(640),
		UIntP:    &uInt,
		UInt8P:   &uInt8,
		UInt16P:  &uInt16,
		UInt32P:  &uInt32,
		UInt64P:  &uInt64,
		Bool:     true,
		BoolP:    &Bool,
		Float32:  32.1,
		Float32P: &f32p,
		Float64:  64.1,
		Float64P: &f64p,
		Byte:     1,
		ByteP:    &Byte,

		String:     "string",
		StringP:    &str,
		Interface:  "interface",
		InterfaceP: &interfaces,

		Struct: struct{}{},

		StructEl: struct {
			Name string `json:"name,select(all|struct)"`
		}(struct{ Name string }{Name: "el"}),

		Map: map[string]interface{}{
			"string": "map val",
			"struct": UsersCase{
				Name: "hhhhh",
			},
		},
		MapP: &map[string]interface{}{
			"map_p": "map val",
		},
		StructP: &struct{}{},
		Structs: UsersCase{
			Name: "name",
			Age:  10,
			Struct: Child{
				CAge:  100,
				CName: "cname",
			},
		},

		StructsP: &UserP{
			Name: &nameP,
			Age:  &ageP,
			Struct: &ChildP{
				CName: &nameP,
				CAge:  &ageP,
			},
		},

		SliceBool:  []bool{Bool, Bool},
		SliceBoolP: []*bool{&Bool},
		SliceByte:  []byte{Byte},
		SliceByteP: []*byte{&Byte},

		SliceInt: []int{
			1, 2,
		},
		SliceInt8: []int8{
			1, 2,
		},
		SliceInt16: []int16{
			1, 2,
		},
		SliceInt32: []int32{
			1, 2,
		},
		SliceInt64: []int64{
			1, 2,
		},
		SliceIntP: []*int{
			&Int, &Int,
		},
		SliceInt8P: []*int8{
			&Int8,
		},
		SliceInt16P: []*int16{
			&Int16,
		},
		SliceInt32I: []*int32{
			&Int32,
		},
		SliceInt64I: []*int64{
			&Int64,
		},
		SliceUint:     []uint{1, 3},
		SliceUint8:    []uint8{1, 2},
		SliceUint16:   []uint16{1, 3},
		SliceUint32:   []uint32{1, 2},
		SliceUint64:   []uint64{1, 4},
		SliceUintP:    []*uint{&uInt},
		SliceUint8P:   []*uint8{&uInt8},
		SliceUint16P:  []*uint16{&uInt16},
		SliceUint32P:  []*uint32{&uInt32},
		SliceUint64P:  []*uint64{&uInt64},
		SliceFloat64:  []float64{12.3},
		SliceFloat64P: []*float64{&f64p},
		SliceFloat32:  []float32{12.7},
		SliceFloat32P: []*float32{&f32p},

		SliceString: []string{
			"slice string", "123",
		},
		SliceStringS: []*string{
			&str, &str,
		},
		SliceInterface: []interface{}{
			"12", "13",
		},
		SliceInterfaceP: []interface{}{
			&str, &str,
		},
		SliceStruct:  []struct{}{},
		SliceStructP: []*struct{}{},

		SliceTime: []time.Time{
			time.Now(), time.Now(),
		},
		SliceUsersCase: []UsersCase{
			{Name: nameP},
		},
		SliceUserP: []*UserP{
			{
				Age: &ageP,
			},
		},
		SliceMap: []map[string]interface{}{
			{
				"map1": 1,
			},
			{
				"map2": 2,
			},
		},
		SliceMapP: []*map[string]interface{}{
			{
				"map1": 1,
			},
			{
				"map2": 2,
			},
		},

		SliceSliceInt: [][]int{
			{1, 23},
			{2, 3},
		},
		SliceSliceUsersCase: [][]UsersCase{
			{
				UsersCase{
					Age: 1,
				},
			},
			{
				UsersCase{
					Age: 2,
				},
			},
		},
	}
	return tests
}

type User struct {
	Name *string `json:"name,select(all)"`
}

var s string

func TestTestCases(t *testing.T) {

	t.Run("all", func(t *testing.T) {
		filter := SelectMarshal("all", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"anonymous_value_name":"anonymous_value_name","article":{"name":"","page_num":0,"page_size":0,"price":"","title":""},"bool":true,"bool_p":true,"byte":1,"byte_p":10,"float32":32.1,"float64":64.1,"float_32_p":320.1,"float_64_p":320.1,"int":100,"int16":16,"int16_p":16,"int32":32,"int32_p":32,"int64":64,"int64_p":64,"int8":8,"int8_p":8,"int_p":100,"interface":"interface","interface_p":"interface p","m_time":"0001-01-01T00:00:00Z","m_time_p":null,"map":{"string":"map val","struct":{"age":0,"name":"hhhhh","struct":{"c_age":0,"c_name":""}}},"map_p":{"map_p":"map val"},"slice_bool":[true,true],"slice_bool_p":[true],"slice_byte":[10],"slice_byte_p":[10],"slice_float_32":[12.7],"slice_float_32_p":[320.1],"slice_float_64":[12.3],"slice_float_64_p":[320.1],"slice_int":[1,2],"slice_int_16":[1,2],"slice_int_16_p":[16],"slice_int_32":[1,2],"slice_int_32_i":[32],"slice_int_64":[1,2],"slice_int_64_i":[64],"slice_int_8":[1,2],"slice_int_8_p":[8],"slice_int_p":[100,100],"slice_interface":["12","13"],"slice_interface_p":["string p","string p"],"slice_map":[{"map1":1},{"map2":2}],"slice_map_p":[{"map1":1},{"map2":2}],"slice_slice_int":[[1,23],[2,3]],"slice_slice_users_case":[[{"age":1,"name":"","struct":{"c_age":0,"c_name":""}}],[{"age":2,"name":"","struct":{"c_age":0,"c_name":""}}]],"slice_string":["slice string","123"],"slice_string_s":["string p","string p"],"slice_struct":[],"slice_struct_p":[],"slice_time":["2022-03-07T16:58:40.546172+08:00","2022-03-07T16:58:40.546172+08:00"],"slice_uint":[1,3],"slice_uint_16":[1,3],"slice_uint_16_p":[16],"slice_uint_32":[1,2],"slice_uint_32_p":[32],"slice_uint_64":[1,4],"slice_uint_64_p":[64],"slice_uint_8":[1,2],"slice_uint_8_p":[8],"slice_uint_p":[100],"slice_user_p":[{"age":10,"name":null,"struct":null}],"slice_users_case":[{"age":0,"name":"nameP","struct":{"c_age":0,"c_name":""}}],"string":"string","string_p":"string p","struct":{},"struct_el":{"name":"el"},"struct_p":{},"structs":{"age":10,"name":"name","struct":{"c_age":100,"c_name":"cname"}},"structs_p":{"age":10,"name":"nameP","struct":{"c_age":10,"c_name":"nameP"}},"time":"0001-01-01T00:00:00Z","time_p":null,"u_int":1000,"u_int16":160,"u_int32":320,"u_int64":640,"u_int8":80,"u_intP":100,"u_int_16_p":16,"u_int_32_p":32,"u_int_64_p":64,"u_int_8_p":8,"y_time":"0001-01-01T00:00:00Z","y_time_p":null}
	})

	t.Run("intAll", func(t *testing.T) {
		filter := SelectMarshal("intAll", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"int":100,"int16":16,"int16_p":16,"int32":32,"int32_p":32,"int64":64,"int64_p":64,"int8":8,"int8_p":8,"int_p":100}
	})

	t.Run("sliceAll", func(t *testing.T) {
		filter := SelectMarshal("sliceAll", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"slice_bool":[true,true],"slice_bool_p":[true],"slice_byte":[10],"slice_byte_p":[10],"slice_float_32":[12.7],"slice_float_32_p":[320.1],"slice_float_64":[12.3],"slice_float_64_p":[320.1],"slice_int":[1,2],"slice_int_16":[1,2],"slice_int_16_p":[16],"slice_int_32":[1,2],"slice_int_32_i":[32],"slice_int_64":[1,2],"slice_int_64_i":[64],"slice_int_8":[1,2],"slice_int_8_p":[8],"slice_int_p":[100,100],"slice_interface":["12","13"],"slice_interface_p":["string p","string p"],"slice_map":[{"map1":1},{"map2":2}],"slice_map_p":[{"map1":1},{"map2":2}],"slice_slice_int":[[1,23],[2,3]],"slice_slice_users_case":[[],[]],"slice_string":["slice string","123"],"slice_string_s":["string p","string p"],"slice_struct":[],"slice_struct_p":[],"slice_time":["2022-03-07T17:03:15.003074+08:00","2022-03-07T17:03:15.003074+08:00"],"slice_uint":[1,3],"slice_uint_16":[1,3],"slice_uint_16_p":[16],"slice_uint_32":[1,2],"slice_uint_32_p":[32],"slice_uint_64":[1,4],"slice_uint_64_p":[64],"slice_uint_8":[1,2],"slice_uint_8_p":[8],"slice_uint_p":[100],"slice_user_p":[],"slice_users_case":[]}
	})

	t.Run("struct", func(t *testing.T) {
		filter := SelectMarshal("struct", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"struct":{},"struct_el":{"name":"el"},"struct_p":{},"structs":{"struct":{"c_age":100,"c_name":"cname"}},"structs_p":{"struct":{"c_age":10}}}
	})
	t.Run("Anonymous", func(t *testing.T) {
		filter := SelectMarshal("Anonymous", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"anonymous_value_name":"anonymous_value_name","article":{"name":"","page_num":0,"page_size":0,"price":"","title":""}}
	})
	t.Run("mapAll", func(t *testing.T) {
		filter := SelectMarshal("mapAll", NewTestCases())
		fmt.Println(filter.MustJSON())
		//{"map":{"string":"map val"},"map_p":{"map_p":"map val"}}
	})

}
