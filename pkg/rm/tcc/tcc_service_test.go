package tcc

import (
	_ "context"
	"testing"
)

func TestTCCServiceProxy_RegisterResource(t1 *testing.T) {

	type fields struct {
		referenceName string
		TCCResource   *TCCResource
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		//{
		//	name:    "zhangsan",
		//	fields:  fields{
		//		referenceName: "test",
		//	}},
		//	wantErr: true,
		//},
	}
	// TODO: Add test cases.

	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &TCCServiceProxy{
				referenceName: tt.fields.referenceName,
				TCCResource:   tt.fields.TCCResource,
			}
			if err := t.RegisterResource(); (err != nil) != tt.wantErr {
				t1.Errorf("RegisterResource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
