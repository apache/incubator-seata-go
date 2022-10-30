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

package flagext

type Secret struct {
	Value string
}

// String implements flag.Value
func (v Secret) String() string {
	return v.Value
}

// Set implements flag.Value
func (v *Secret) Set(s string) error {
	v.Value = s
	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (v *Secret) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	return v.Set(s)
}

// MarshalYAML implements yaml.Marshaler.
func (v Secret) MarshalYAML() (interface{}, error) {
	if len(v.Value) == 0 {
		return "", nil
	}
	return "********", nil
}
