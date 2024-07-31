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

package xa

import (
	"database/sql/driver"
	"testing"
	"time"
)

func TestOracleXAConn_Commit(t *testing.T) {
	type fields struct {
		Conn driver.Conn
	}
	type args struct {
		xid      string
		onePhase bool
	}
	var tests []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: tt.fields.Conn,
			}
			if err := c.Commit(tt.args.xid, tt.args.onePhase); (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_End(t *testing.T) {
	type fields struct {
		Conn driver.Conn
	}
	type args struct {
		xid   string
		flags int
	}
	var tests []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: tt.fields.Conn,
			}
			if err := c.End(tt.args.xid, tt.args.flags); (err != nil) != tt.wantErr {
				t.Errorf("End() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_Forget(t *testing.T) {
	type fields struct {
		Conn driver.Conn
	}
	type args struct {
		xid string
	}
	var tests []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: tt.fields.Conn,
			}
			if err := c.Forget(tt.args.xid); (err != nil) != tt.wantErr {
				t.Errorf("Forget() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOracleXAConn_GetTransactionTimeout(t *testing.T) {
	type fields struct {
		Conn driver.Conn
	}
	var tests []struct {
		name   string
		fields fields
		want   time.Duration
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &OracleXAConn{
				Conn: tt.fields.Conn,
			}
			if got := c.GetTransactionTimeout(); got != tt.want {
				t.Errorf("GetTransactionTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}
