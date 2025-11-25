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

package convert

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAsString(t *testing.T) {
	tests := []struct {
		name string
		arg  interface{}
		want string
	}{
		{
			name: "string",
			arg:  "hello",
			want: "hello",
		},
		{
			name: "[]byte",
			arg:  []byte("world"),
			want: "world",
		},
		{
			name: "int",
			arg:  123,
			want: "123",
		},
		{
			name: "int8",
			arg:  int8(123),
			want: "123",
		},
		{
			name: "int16",
			arg:  int16(123),
			want: "123",
		},
		{
			name: "int32",
			arg:  int32(123),
			want: "123",
		},
		{
			name: "int64",
			arg:  int64(123),
			want: "123",
		},
		{
			name: "uint",
			arg:  uint(123),
			want: "123",
		},
		{
			name: "uint8",
			arg:  uint8(123),
			want: "123",
		},
		{
			name: "uint16",
			arg:  uint16(123),
			want: "123",
		},
		{
			name: "uint32",
			arg:  uint32(123),
			want: "123",
		},
		{
			name: "uint64",
			arg:  uint64(123),
			want: "123",
		},
		{
			name: "float32",
			arg:  float32(1.23),
			want: "1.23",
		},
		{
			name: "float64",
			arg:  float64(1.23),
			want: "1.23",
		},
		{
			name: "bool true",
			arg:  true,
			want: "true",
		},
		{
			name: "bool false",
			arg:  false,
			want: "false",
		},
		{
			name: "other type",
			arg:  struct{ Name string }{Name: "test"},
			want: "{test}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := asString(tt.arg); got != tt.want {
				t.Errorf("asString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAsBytes(t *testing.T) {
	tests := []struct {
		name    string
		arg     interface{}
		want    []byte
		wantErr bool
	}{
		{
			name: "string",
			arg:  "hello",
			want: []byte("hello"),
		},
		{
			name: "int",
			arg:  123,
			want: []byte("123"),
		},
		{
			name: "int8",
			arg:  int8(123),
			want: []byte("123"),
		},
		{
			name: "int16",
			arg:  int16(123),
			want: []byte("123"),
		},
		{
			name: "int32",
			arg:  int32(123),
			want: []byte("123"),
		},
		{
			name: "int64",
			arg:  int64(123),
			want: []byte("123"),
		},
		{
			name: "uint",
			arg:  uint(123),
			want: []byte("123"),
		},
		{
			name: "uint8",
			arg:  uint8(123),
			want: []byte("123"),
		},
		{
			name: "uint16",
			arg:  uint16(123),
			want: []byte("123"),
		},
		{
			name: "uint32",
			arg:  uint32(123),
			want: []byte("123"),
		},
		{
			name: "uint64",
			arg:  uint64(123),
			want: []byte("123"),
		},
		{
			name: "float32",
			arg:  float32(1.23),
			want: []byte("1.23"),
		},
		{
			name: "float64",
			arg:  float64(1.23),
			want: []byte("1.23"),
		},
		{
			name: "bool true",
			arg:  true,
			want: []byte("true"),
		},
		{
			name: "bool false",
			arg:  false,
			want: []byte("false"),
		},
		{
			name:    "unsupported type",
			arg:     make(chan int),
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := asBytes(nil, reflect.ValueOf(tt.arg))
			if (err != true) != tt.wantErr {
				t.Errorf("asBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertAssignRows_BasicTypes(t *testing.T) {
	var s string
	if err := ConvertAssignRows(&s, "hello"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "hello" {
		t.Errorf("expected hello, got %s", s)
	}

	var b []byte
	if err := ConvertAssignRows(&b, []byte("abc")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(b) != "abc" {
		t.Errorf("expected abc, got %s", b)
	}
}

func TestConvertAssignRows_TimeAndNil(t *testing.T) {
	now := time.Now()
	var s string
	if err := ConvertAssignRows(&s, now); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(s, "T") {
		t.Errorf("expected RFC3339 format, got %s", s)
	}

	var ptr *[]byte
	err := ConvertAssignRows(ptr, "abc")
	if err == nil {
		t.Error("expected errNilPtr, got nil")
	}

	var iface interface{}
	if err := ConvertAssignRows(&iface, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if iface != nil {
		t.Errorf("expected nil interface, got %v", iface)
	}
}

func TestConvertAssignRows_NumberConversions(t *testing.T) {
	var i int64
	if err := ConvertAssignRows(&i, "42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if i != 42 {
		t.Errorf("expected 42, got %d", i)
	}

	var f float64
	if err := ConvertAssignRows(&f, "3.14"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != 3.14 {
		t.Errorf("expected 3.14, got %v", f)
	}

	var u uint
	if err := ConvertAssignRows(&u, "123"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertAssignRows_ErrorsAndEdgeCases(t *testing.T) {
	err := ConvertAssignRows(123, "abc")
	if err == nil {
		t.Error("expected error for non-pointer dest")
	}

	var ch chan int
	err = ConvertAssignRows(&ch, "abc")
	if err == nil {
		t.Error("expected unsupported type error")
	}

	ne := &strconv.NumError{Func: "ParseInt", Num: "abc", Err: strconv.ErrSyntax}
	if got := strconvErr(ne); got != strconv.ErrSyntax {
		t.Errorf("expected ErrSyntax, got %v", got)
	}

	src := []byte("data")
	cloned := cloneBytes(src)
	if &src[0] == &cloned[0] {
		t.Error("cloneBytes() did not create a copy")
	}
}

// Additional comprehensive tests to increase coverage

func TestCloneBytes(t *testing.T) {
	t.Run("nil bytes", func(t *testing.T) {
		result := cloneBytes(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("empty bytes", func(t *testing.T) {
		result := cloneBytes([]byte{})
		if result == nil || len(result) != 0 {
			t.Errorf("expected empty slice, got %v", result)
		}
	})

	t.Run("non-empty bytes", func(t *testing.T) {
		src := []byte("hello")
		result := cloneBytes(src)
		if string(result) != "hello" {
			t.Errorf("expected 'hello', got %v", string(result))
		}
		// Ensure it's a copy
		if len(src) > 0 && len(result) > 0 && &src[0] == &result[0] {
			t.Error("expected a copy, got same reference")
		}
	})
}

func TestStrconvErr(t *testing.T) {
	t.Run("NumError", func(t *testing.T) {
		numErr := &strconv.NumError{
			Func: "ParseInt",
			Num:  "abc",
			Err:  strconv.ErrSyntax,
		}
		result := strconvErr(numErr)
		if result != strconv.ErrSyntax {
			t.Errorf("expected ErrSyntax, got %v", result)
		}
	})

	t.Run("regular error", func(t *testing.T) {
		regularErr := errors.New("some error")
		result := strconvErr(regularErr)
		if result != regularErr {
			t.Errorf("expected same error, got %v", result)
		}
	}
}

// Mock Scanner for testing
type mockScanner struct {
	value interface{}
	err   error
}

func (m *mockScanner) Scan(src interface{}) error {
	if m.err != nil {
		return m.err
	}
	m.value = src
	return nil
}

// Mock decimal types
type mockDecimal struct {
	form        byte
	negative    bool
	coefficient []byte
	exponent    int32
}

func (m *mockDecimal) Decompose(buf []byte) (form byte, negative bool, coefficient []byte, exponent int32) {
	return m.form, m.negative, m.coefficient, m.exponent
}

func (m *mockDecimal) Compose(form byte, negative bool, coefficient []byte, exponent int32) error {
	m.form = form
	m.negative = negative
	m.coefficient = coefficient
	m.exponent = exponent
	return nil
}

func TestConvertAssignRows_StringConversions(t *testing.T) {
	t.Run("string to nil string pointer", func(t *testing.T) {
		var dest *string
		err := ConvertAssignRows(dest, "hello")
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("string to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "hello" {
			t.Errorf("expected 'hello', got %v", string(dest))
		}
	})

	t.Run("string to nil []byte pointer", func(t *testing.T) {
		var dest *[]byte
		err := ConvertAssignRows(dest, "hello")
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("string to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, "hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "hello" {
			t.Errorf("expected 'hello', got %v", string(dest))
		}
	})

	t.Run("string to nil RawBytes pointer", func(t *testing.T) {
		var dest *RawBytes
		err := ConvertAssignRows(dest, "hello")
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})
}

func TestConvertAssignRows_ByteSliceConversions(t *testing.T) {
	t.Run("[]byte to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, []byte("world"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "world" {
			t.Errorf("expected 'world', got %v", dest)
		}
	})

	t.Run("[]byte to nil string pointer", func(t *testing.T) {
		var dest *string
		err := ConvertAssignRows(dest, []byte("world"))
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("[]byte to interface", func(t *testing.T) {
		var dest interface{}
		src := []byte("test")
		err := ConvertAssignRows(&dest, src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		destBytes, ok := dest.([]byte)
		if !ok {
			t.Errorf("expected []byte, got %T", dest)
		}
		// Should be a clone
		if len(src) > 0 && len(destBytes) > 0 && &src[0] == &destBytes[0] {
			t.Error("expected cloned bytes, got same reference")
		}
	})

	t.Run("[]byte to nil interface pointer", func(t *testing.T) {
		var dest *interface{}
		err := ConvertAssignRows(dest, []byte("test"))
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("[]byte to []byte", func(t *testing.T) {
		var dest []byte
		src := []byte("copy")
		err := ConvertAssignRows(&dest, src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "copy" {
			t.Errorf("expected 'copy', got %v", string(dest))
		}
		// Should be cloned
		if len(src) > 0 && len(dest) > 0 && &src[0] == &dest[0] {
			t.Error("expected cloned bytes, got same reference")
		}
	})

	t.Run("[]byte to nil []byte pointer", func(t *testing.T) {
		var dest *[]byte
		err := ConvertAssignRows(dest, []byte("copy"))
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("[]byte to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, []byte("raw"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "raw" {
			t.Errorf("expected 'raw', got %v", string(dest))
		}
	})

	t.Run("[]byte to nil RawBytes pointer", func(t *testing.T) {
		var dest *RawBytes
		err := ConvertAssignRows(dest, []byte("raw"))
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})
}

func TestConvertAssignRows_TimeConversions(t *testing.T) {
	now := time.Now()

	t.Run("time.Time to time.Time", func(t *testing.T) {
		var dest time.Time
		err := ConvertAssignRows(&dest, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dest.Equal(now) {
			t.Errorf("times not equal")
		}
	})

	t.Run("time.Time to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := now.Format(time.RFC3339Nano)
		if dest != expected {
			t.Errorf("expected %v, got %v", expected, dest)
		}
	})

	t.Run("time.Time to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := now.Format(time.RFC3339Nano)
		if string(dest) != expected {
			t.Errorf("expected %v, got %v", expected, string(dest))
		}
	})

	t.Run("time.Time to nil []byte pointer", func(t *testing.T) {
		var dest *[]byte
		err := ConvertAssignRows(dest, now)
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})

	t.Run("time.Time to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := now.Format(time.RFC3339Nano)
		if string(dest) != expected {
			t.Errorf("expected %v, got %v", expected, string(dest))
		}
	})

	t.Run("time.Time to nil RawBytes pointer", func(t *testing.T) {
		var dest *RawBytes
		err := ConvertAssignRows(dest, now)
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})
}

func TestConvertAssignRows_DecimalTypes(t *testing.T) {
	t.Run("decimal compose/decompose", func(t *testing.T) {
		src := &mockDecimal{
			form:        1,
			negative:    true,
			coefficient: []byte{1, 2, 3},
			exponent:    2,
		}
		dest := &mockDecimal{}
		err := ConvertAssignRows(dest, src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest.form != 1 || !dest.negative || dest.exponent != 2 {
			t.Error("decimal not composed correctly")
		}
	})
}

func TestConvertAssignRows_NilConversions(t *testing.T) {
	t.Run("nil to interface", func(t *testing.T) {
		var dest interface{}
		err := ConvertAssignRows(&dest, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != nil {
			t.Errorf("expected nil, got %v", dest)
		}
	})

	t.Run("nil to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != nil {
			t.Errorf("expected nil, got %v", dest)
		}
	})

	t.Run("nil to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != nil {
			t.Errorf("expected nil, got %v", dest)
		}
	})
}

func TestConvertAssignRows_ScannerInterface(t *testing.T) {
	t.Run("scanner success", func(t *testing.T) {
		scanner := &mockScanner{}
		err := ConvertAssignRows(scanner, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if scanner.value != "test" {
			t.Errorf("expected 'test', got %v", scanner.value)
		}
	})

	t.Run("scanner error", func(t *testing.T) {
		expectedErr := errors.New("scan error")
		scanner := &mockScanner{err: expectedErr}
		err := ConvertAssignRows(scanner, "test")
		if err != expectedErr {
			t.Errorf("expected %v, got %v", expectedErr, err)
		}
	})
}

func TestConvertAssignRows_ReflectStringConversions(t *testing.T) {
	t.Run("int to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, 123)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "123" {
			t.Errorf("expected '123', got %v", dest)
		}
	})

	t.Run("bool to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "true" {
			t.Errorf("expected 'true', got %v", dest)
		}
	})

	t.Run("float to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, 1.23)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "1.23" {
			t.Errorf("expected '1.23', got %v", dest)
		}
	})

	t.Run("uint to string", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(&dest, uint(456))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "456" {
			t.Errorf("expected '456', got %v", dest)
		}
	})
}

func TestConvertAssignRows_ReflectBytesConversions(t *testing.T) {
	t.Run("int to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, 456)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "456" {
			t.Errorf("expected '456', got %v", string(dest))
		}
	})

	t.Run("bool to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "false" {
			t.Errorf("expected 'false', got %v", string(dest))
		}
	})

	t.Run("float to []byte", func(t *testing.T) {
		var dest []byte
		err := ConvertAssignRows(&dest, 3.14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "3.14" {
			t.Errorf("expected '3.14', got %v", string(dest))
		}
	})
}

func TestConvertAssignRows_RawBytesConversions(t *testing.T) {
	t.Run("int to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, 789)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "789" {
			t.Errorf("expected '789', got %v", string(dest))
		}
	})

	t.Run("bool to RawBytes", func(t *testing.T) {
		var dest RawBytes
		err := ConvertAssignRows(&dest, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(dest) != "true" {
			t.Errorf("expected 'true', got %v", string(dest))
		}
	})
}

func TestConvertAssignRows_BoolConversions(t *testing.T) {
	t.Run("bool to bool", func(t *testing.T) {
		var dest bool
		err := ConvertAssignRows(&dest, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dest {
			t.Error("expected true")
		}
	})

	t.Run("int to bool", func(t *testing.T) {
		var dest bool
		err := ConvertAssignRows(&dest, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dest {
			t.Error("expected true")
		}
	})

	t.Run("string to bool", func(t *testing.T) {
		var dest bool
		err := ConvertAssignRows(&dest, "true")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dest {
			t.Error("expected true")
		}
	})
}

func TestConvertAssignRows_InterfaceConversions(t *testing.T) {
	t.Run("string to interface", func(t *testing.T) {
		var dest interface{}
		err := ConvertAssignRows(&dest, "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "test" {
			t.Errorf("expected 'test', got %v", dest)
		}
	})

	t.Run("int to interface", func(t *testing.T) {
		var dest interface{}
		err := ConvertAssignRows(&dest, 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != 42 {
			t.Errorf("expected 42, got %v", dest)
		}
	})
}

func TestConvertAssignRows_ErrorCases(t *testing.T) {
	t.Run("non-pointer dest", func(t *testing.T) {
		var dest string
		err := ConvertAssignRows(dest, "test")
		if err == nil || !strings.Contains(err.Error(), "not a pointer") {
			t.Errorf("expected 'not a pointer' error, got %v", err)
		}
	})

	t.Run("nil pointer dest", func(t *testing.T) {
		var dest *string
		err := ConvertAssignRows(dest, "test")
		if err != errNilPtr {
			t.Errorf("expected errNilPtr, got %v", err)
		}
	})
}

func TestConvertAssignRows_AssignableTypes(t *testing.T) {
	type MyString string

	t.Run("assignable custom type", func(t *testing.T) {
		var dest MyString
		err := ConvertAssignRows(&dest, MyString("custom"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dest != "custom" {
			t.Errorf("expected 'custom', got %v", dest)
		}
	})

	t.Run("[]byte assignable with cloning", func(t *testing.T) {
		var dest []byte
		src := []byte("assign")
		err := ConvertAssignRows(&dest, src)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should be cloned
		if len(src) > 0 && len(dest) > 0 && &src[0] == &dest[0] {
			t.Error("expected cloned bytes")
		}
	})
}