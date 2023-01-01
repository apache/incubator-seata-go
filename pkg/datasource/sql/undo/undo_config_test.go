package undo

import (
	"flag"
	"testing"
)

func TestCompressConfig_RegisterFlagsWithPrefix(t *testing.T) {
	type fields struct {
		Enable    bool
		Type      string
		Threshold int
	}
	type args struct {
		prefix string
		f      *flag.FlagSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestUndoConfig_RegisterFlagsWithPrefix(t *testing.T) {
	type fields struct {
		DataValidation        bool
		LogSerialization      string
		LogTable              string
		OnlyCareUpdateColumns bool
		Compress              CompressConfig
	}
	type args struct {
		prefix string
		f      *flag.FlagSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
