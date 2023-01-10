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

package util

import "database/sql/driver"

func NamedValueToValue(nvs []driver.NamedValue) []driver.Value {
	vs := make([]driver.Value, 0, len(nvs))
	for _, nv := range nvs {
		vs = append(vs, nv.Value)
	}
	return vs
}

func ValueToNamedValue(vs []driver.Value) []driver.NamedValue {
	nvs := make([]driver.NamedValue, 0, len(vs))
	for i, v := range vs {
		nvs = append(nvs, driver.NamedValue{
			Value:   v,
			Ordinal: i,
		})
	}
	return nvs
}
