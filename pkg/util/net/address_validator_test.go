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

package net

import (
	"reflect"
	"testing"
)

func TestAddressValidator(t *testing.T) {
	type IPAddr struct {
		host string
		port int
	}
	tests := []struct {
		name       string
		address    string
		want       *IPAddr
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:    "normal single endpoint.",
			address: "127.0.0.1:8091",
			want: &IPAddr{
				"127.0.0.1", 8091,
			},
			wantErr: false,
		},
		{
			name:       "addr is empty.",
			address:    "",
			want:       nil,
			wantErr:    true,
			wantErrMsg: "split ip err: param addr must not empty",
		},
		{
			name:       "format is not ip:port",
			address:    "127.0.0.18091",
			want:       nil,
			wantErr:    true,
			wantErrMsg: "address 127.0.0.18091: missing port in address",
		},
		{
			name:       "port is not number",
			address:    "127.0.0.1:abc",
			want:       nil,
			wantErr:    true,
			wantErrMsg: "strconv.Atoi: parsing \"abc\": invalid syntax",
		},
		{
			name:    "endpoint is ipv6",
			address: "[2000:0000:0000:0000:0001:2345:6789:abcd]:8080",
			want: &IPAddr{
				"2000:0000:0000:0000:0001:2345:6789:abcd", 8080,
			},
			wantErr: false,
		},
		{
			name:    "endpoint is ipv6",
			address: "[2000:0000:0000:0000:0001:2345:6789:abcd%10]:8080",
			want: &IPAddr{
				"2000:0000:0000:0000:0001:2345:6789:abcd", 8080,
			},
			wantErr: false,
		},
		{
			name:    "endpoint is ipv6",
			address: "2000:0000:0000:0000:0001:2345:6789:abcd:8080",
			want: &IPAddr{
				"2000:0000:0000:0000:0001:2345:6789:abcd", 8080,
			},
			wantErr: false,
		},
		{
			name:    "endpoint is ipv6",
			address: "[::]:8080",
			want: &IPAddr{
				"::", 8080,
			},
			wantErr: false,
		},
		{
			name:    "endpoint is ipv6",
			address: "::FFFF:192.168.1.2:8080",
			want: &IPAddr{
				"::FFFF:192.168.1.2", 8080,
			},
			wantErr: false,
		},
		{
			name:    "endpoint is ipv6",
			address: "[::FFFF:192.168.1.2]:8080",
			want: &IPAddr{
				"::FFFF:192.168.1.2", 8080,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := SplitIPPortStr(tt.address)

			if (err != nil) != tt.wantErr {
				t.Errorf("SplitIPPortStr() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("SplitIPPortStr() errMsg = %v, wantErrMsg = %v", err.Error(), tt.wantErrMsg)
			}
			got := &IPAddr{host, port}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitIPPortStr() got = %v, want = %v", got, tt.want)
			}
		})
	}
}
