package conn

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
