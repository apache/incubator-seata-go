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

package parser

import (
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// ConfigurationParser interface
type ConfigurationParser interface {
	Parse(string) (map[string]string, error)
}

// LoadYMLConfig Load yml config byte from file check file type is *.yml or *.yaml`
func LoadYMLConfig(confProFile string) ([]byte, error) {
	if len(confProFile) == 0 {
		return nil, errors.Errorf("application configure(provider) file name is nil")
	}

	if path.Ext(confProFile) != ".yml" && path.Ext(confProFile) != ".yaml" {
		return nil, errors.Errorf("application configure file name{%v} suffix must be .yml or .yaml", confProFile)
	}

	return ioutil.ReadFile(confProFile)
}

// UnmarshalYML unmarshals decodes the first document found within the in byte slice and assigns decoded values into the out value.
func UnmarshalYML(data []byte, out interface{}) error {
	return yaml.Unmarshal(data, out)
}
