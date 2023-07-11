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

package discovery

import (
	"reflect"
	"testing"
)

func TestFileRegistryService_Lookup(t *testing.T) {
	type fields struct {
		serviceConfig *ServiceConfig
	}
	type args struct {
		key string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       []*ServiceInstance
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "normal single endpoint.",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
					Grouplist: map[string]string{
						"default": "127.0.0.1:8091",
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "127.0.0.1",
					Port: 8091,
				},
			},
			wantErr: false,
		},
		{
			name: "normal multi endpoints.",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
					Grouplist: map[string]string{
						"default": "127.0.0.1:8091;192.168.0.1:8092",
					},
				},
			},
			want: []*ServiceInstance{
				{
					Addr: "127.0.0.1",
					Port: 8091,
				},
				{
					Addr: "192.168.0.1",
					Port: 8092,
				},
			},
			wantErr: false,
		},
		{
			name: "vgroup is empty.",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "",
					},
				},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "vgroup is empty. key: default_tx_group",
		},
		{
			name: "endpoint is empty.",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
				},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "endpoint is empty. key: default_tx_group group: default",
		},
		{
			name: "format is not ip:port",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
					Grouplist: map[string]string{
						"default": "127.0.0.18091",
					},
				},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "endpoint format should like ip:port. endpoint: 127.0.0.18091",
		},
		{
			name: "port is not number",
			args: args{
				key: "default_tx_group",
			},
			fields: fields{
				serviceConfig: &ServiceConfig{
					VgroupMapping: map[string]string{
						"default_tx_group": "default",
					},
					Grouplist: map[string]string{
						"default": "127.0.0.1:abc",
					},
				},
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: "strconv.Atoi: parsing \"abc\": invalid syntax",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FileRegistryService{
				serviceConfig: tt.fields.serviceConfig,
			}
			got, err := s.Lookup(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != tt.wantErrMsg {
				t.Errorf("Lookup() errMsg = %v, wantErrMsg = %v", err.Error(), tt.wantErrMsg)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Lookup() got = %v, want = %v", got, tt.want)
			}
		})
	}
}
