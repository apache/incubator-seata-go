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
	"reflect"
	"testing"
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
