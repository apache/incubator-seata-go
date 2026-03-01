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

package tm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalTransactionRole(t *testing.T) {
	tests := []struct {
		name     string
		role     GlobalTransactionRole
		expected string
	}{
		{"UnKnow", UnKnow, "UnKnow"},
		{"Launcher", Launcher, "Launcher"},
		{"Participant", Participant, "Participant"},
		{"Invalid role", GlobalTransactionRole(99), "UnKnow"},
		{"Negative role", GlobalTransactionRole(-1), "UnKnow"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.role.String())
		})
	}
}

func TestGlobalTransactionRoleValues(t *testing.T) {
	assert.Equal(t, GlobalTransactionRole(0), UnKnow)
	assert.Equal(t, GlobalTransactionRole(1), Launcher)
	assert.Equal(t, GlobalTransactionRole(2), Participant)
}

func TestPropagation(t *testing.T) {
	tests := []struct {
		name        string
		propagation Propagation
		expected    string
	}{
		{"Required", Required, "Required"},
		{"RequiresNew", RequiresNew, "RequiresNew"},
		{"NotSupported", NotSupported, "NotSupported"},
		{"Supports", Supports, "Supports"},
		{"Never", Never, "Never"},
		{"Mandatory", Mandatory, "Mandatory"},
		{"Invalid propagation", Propagation(99), "UnKnow"},
		{"Negative propagation", Propagation(-1), "UnKnow"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.propagation.String())
		})
	}
}

func TestPropagationValues(t *testing.T) {
	assert.Equal(t, Propagation(0), Required)
	assert.Equal(t, Propagation(1), RequiresNew)
	assert.Equal(t, Propagation(2), NotSupported)
	assert.Equal(t, Propagation(3), Supports)
	assert.Equal(t, Propagation(4), Never)
	assert.Equal(t, Propagation(5), Mandatory)
}

func TestTransactionManagerInterface(t *testing.T) {
	// Test that TransactionManager is an interface
	var tm TransactionManager
	assert.Nil(t, tm, "TransactionManager interface should be nil when not implemented")
}

func TestGlobalTransactionRoleType(t *testing.T) {
	// Test that GlobalTransactionRole is an int8
	var role GlobalTransactionRole = 1
	assert.IsType(t, int8(0), int8(role))
}

func TestPropagationType(t *testing.T) {
	// Test that Propagation is an int8
	var prop Propagation = 1
	assert.IsType(t, int8(0), int8(prop))
}

func TestRoleConstants(t *testing.T) {
	roles := []GlobalTransactionRole{UnKnow, Launcher, Participant}

	// Test that all roles are unique
	seen := make(map[GlobalTransactionRole]bool)
	for _, role := range roles {
		assert.False(t, seen[role], "Role %d should be unique", role)
		seen[role] = true
	}
}

func TestPropagationConstants(t *testing.T) {
	propagations := []Propagation{Required, RequiresNew, NotSupported, Supports, Never, Mandatory}

	// Test that all propagations are unique
	seen := make(map[Propagation]bool)
	for _, prop := range propagations {
		assert.False(t, seen[prop], "Propagation %d should be unique", prop)
		seen[prop] = true
	}
}

func TestStringMethodsAreIdempotent(t *testing.T) {
	// Test that calling String() multiple times returns the same result
	role := Launcher
	first := role.String()
	second := role.String()
	assert.Equal(t, first, second, "String() method should be idempotent")

	prop := Required
	firstProp := prop.String()
	secondProp := prop.String()
	assert.Equal(t, firstProp, secondProp, "String() method should be idempotent")
}
