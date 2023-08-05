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
package reflectx

import "testing"

type ReferencedServiceImpl struct{}

func (r *ReferencedServiceImpl) Reference() string {
	return "ReferencedServiceImpl"
}

type TestStruct struct{}

func TestGetReference(t *testing.T) {
	s := &ReferencedServiceImpl{}
	ref := GetReference(s)
	if ref != "ReferencedServiceImpl" {
		t.Errorf("GetReference() = %s, want ReferencedServiceImpl", ref)
	}

	ts := TestStruct{}
	ref = GetReference(ts)
	if ref != "TestStruct" {
		t.Errorf("GetReference() = %s, want TestStruct", ref)
	}

	tp := &TestStruct{}
	ref = GetReference(tp)
	if ref != "TestStruct" {
		t.Errorf("GetReference() = %s, want TestStruct", ref)
	}

	var v interface{} = &TestStruct{}
	ref = GetReference(v)
	if ref != "TestStruct" {
		t.Errorf("GetReference() = %s, want TestStruct", ref)
	}
}
