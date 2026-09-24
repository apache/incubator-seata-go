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

import "testing"

func TestRegistryProviderForSupportedTypes(t *testing.T) {
	tests := []struct {
		name         string
		registryType string
	}{
		{name: "file", registryType: FILE},
		{name: "etcd alias", registryType: ETCD},
		{name: "etcd3", registryType: ETCD3},
		{name: "raft", registryType: RAFT},
		{name: "namingserver", registryType: NAMINGSERVER},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, ok := registryProviderFor(tt.registryType)
			if !ok {
				t.Fatalf("provider not found for type %s", tt.registryType)
			}
			if provider == nil {
				t.Fatalf("provider is nil for type %s", tt.registryType)
			}
		})
	}
}

func TestRegistryProviderForUnsupportedType(t *testing.T) {
	provider, ok := registryProviderFor("unknown")
	if ok {
		t.Fatalf("unexpected provider for unknown type: %v", provider)
	}

	err := unsupportedRegistryTypeError("unknown")
	if err.Error() != "service registry not support registry type:unknown" {
		t.Fatalf("unexpected error: %v", err)
	}
}
