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

package constant

import (
	"testing"
)

func TestConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ActionStartTime", ActionStartTime, "action-start-time"},
		{"HostName", HostName, "host-name"},
		{"ActionContext", ActionContext, "actionContext"},
		{"PrepareMethod", PrepareMethod, "sys::prepare"},
		{"CommitMethod", CommitMethod, "sys::commit"},
		{"RollbackMethod", RollbackMethod, "sys::rollback"},
		{"ActionName", ActionName, "actionName"},
		{"SeataXidKey", SeataXidKey, "SEATA_XID"},
		{"XidKey", XidKey, "TX_XID"},
		{"XidKeyLowercase", XidKeyLowercase, "tx_xid"},
		{"MdcXidKey", MdcXidKey, "X-TX-XID"},
		{"MdcBranchIDKey", MdcBranchIDKey, "X-TX-BRANCH-ID"},
		{"BranchTypeKey", BranchTypeKey, "TX_BRANCH_TYPE"},
		{"GlobalLockKey", GlobalLockKey, "TX_LOCK"},
		{"SeataFilterKey", SeataFilterKey, "seataDubboFilter"},
		{"SeataVersion", SeataVersion, "1.1.0"},
		{"TccBusinessActionContextParameter", TccBusinessActionContextParameter, "tccParam"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.constant != test.expected {
				t.Errorf("Expected %s to be %s, but got %s", test.name, test.expected, test.constant)
			}
		})
	}
}

func TestConstantTypes(t *testing.T) {
	// Test that constants are of string type
	constants := []interface{}{
		ActionStartTime,
		HostName,
		ActionContext,
		PrepareMethod,
		CommitMethod,
		RollbackMethod,
		ActionName,
		SeataXidKey,
		XidKey,
		XidKeyLowercase,
		MdcXidKey,
		MdcBranchIDKey,
		BranchTypeKey,
		GlobalLockKey,
		SeataFilterKey,
		SeataVersion,
		TccBusinessActionContextParameter,
	}

	for i, constant := range constants {
		if _, ok := constant.(string); !ok {
			t.Errorf("Constant at index %d is not of string type", i)
		}
	}
}

func TestConstantValues(t *testing.T) {
	// Test that constants are not empty
	constants := map[string]string{
		"ActionStartTime":                   ActionStartTime,
		"HostName":                          HostName,
		"ActionContext":                     ActionContext,
		"PrepareMethod":                     PrepareMethod,
		"CommitMethod":                      CommitMethod,
		"RollbackMethod":                    RollbackMethod,
		"ActionName":                        ActionName,
		"SeataXidKey":                       SeataXidKey,
		"XidKey":                            XidKey,
		"XidKeyLowercase":                   XidKeyLowercase,
		"MdcXidKey":                         MdcXidKey,
		"MdcBranchIDKey":                    MdcBranchIDKey,
		"BranchTypeKey":                     BranchTypeKey,
		"GlobalLockKey":                     GlobalLockKey,
		"SeataFilterKey":                    SeataFilterKey,
		"SeataVersion":                      SeataVersion,
		"TccBusinessActionContextParameter": TccBusinessActionContextParameter,
	}

	for name, value := range constants {
		if value == "" {
			t.Errorf("Constant %s should not be empty", name)
		}
	}
}

func TestMethodConstants(t *testing.T) {
	// Test that method constants follow expected format
	methodConstants := []string{PrepareMethod, CommitMethod, RollbackMethod}
	expectedPrefix := "sys::"

	for _, method := range methodConstants {
		if len(method) <= len(expectedPrefix) || method[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("Method constant %s should start with %s", method, expectedPrefix)
		}
	}
}

func TestXidConstants(t *testing.T) {
	// Test XID related constants
	xidConstants := map[string]string{
		"SeataXidKey":     SeataXidKey,
		"XidKey":          XidKey,
		"XidKeyLowercase": XidKeyLowercase,
		"MdcXidKey":       MdcXidKey,
		"MdcBranchIDKey":  MdcBranchIDKey,
	}

	for name, value := range xidConstants {
		if value == "" {
			t.Errorf("XID constant %s should not be empty", name)
		}
		if name == "XidKeyLowercase" && value != "tx_xid" {
			t.Errorf("XidKeyLowercase should be lowercase: %s", value)
		}
	}
}
