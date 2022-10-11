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

import "reflect"

type ReferencedService interface {
	Reference() string
}

// GetReference return the reference id of the service.
// If the service implemented the ReferencedService interface,
// it will call the Reference method. If not, it will
// return the struct name as the reference id.
func GetReference(service interface{}) string {
	if s, ok := service.(ReferencedService); ok {
		return s.Reference()
	}
	ref := ""
	sType := reflect.TypeOf(service)
	kind := sType.Kind()
	switch kind {
	case reflect.Struct:
		ref = sType.Name()
	case reflect.Ptr:
		sName := sType.Elem().Name()
		if sName != "" {
			ref = sName
		} else {
			ref = sType.Elem().Field(0).Name
		}
	}
	return ref
}
