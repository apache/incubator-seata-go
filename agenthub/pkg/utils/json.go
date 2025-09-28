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

package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// JSONUtils provides JSON serialization utilities
type JSONUtils struct{}

// NewJSONUtils creates a new JSON utilities instance
func NewJSONUtils() *JSONUtils {
	return &JSONUtils{}
}

// Marshal marshals data to JSON with proper formatting
func (j *JSONUtils) Marshal(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// MarshalIndent marshals data to JSON with indentation
func (j *JSONUtils) MarshalIndent(data interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(data, prefix, indent)
}

// Unmarshal unmarshals JSON data
func (j *JSONUtils) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// UnmarshalFromReader unmarshals JSON from a reader
func (j *JSONUtils) UnmarshalFromReader(reader io.Reader, v interface{}) error {
	decoder := json.NewDecoder(reader)
	return decoder.Decode(v)
}

// MarshalToWriter marshals data to a writer
func (j *JSONUtils) MarshalToWriter(writer io.Writer, data interface{}) error {
	encoder := json.NewEncoder(writer)
	return encoder.Encode(data)
}

// MarshalToFile marshals data to a file
func (j *JSONUtils) MarshalToFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// UnmarshalFromFile unmarshals data from a file
func (j *JSONUtils) UnmarshalFromFile(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(v)
}

// YAMLUtils provides YAML serialization utilities
type YAMLUtils struct{}

// NewYAMLUtils creates a new YAML utilities instance
func NewYAMLUtils() *YAMLUtils {
	return &YAMLUtils{}
}

// Marshal marshals data to YAML
func (y *YAMLUtils) Marshal(data interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}

// Unmarshal unmarshals YAML data
func (y *YAMLUtils) Unmarshal(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

// UnmarshalFromReader unmarshals YAML from a reader
func (y *YAMLUtils) UnmarshalFromReader(reader io.Reader, v interface{}) error {
	decoder := yaml.NewDecoder(reader)
	return decoder.Decode(v)
}

// MarshalToWriter marshals data to a writer as YAML
func (y *YAMLUtils) MarshalToWriter(writer io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(writer)
	defer encoder.Close()
	return encoder.Encode(data)
}

// MarshalToFile marshals data to a YAML file
func (y *YAMLUtils) MarshalToFile(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filename, err)
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	return encoder.Encode(data)
}

// UnmarshalFromFile unmarshals data from a YAML file
func (y *YAMLUtils) UnmarshalFromFile(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	return decoder.Decode(v)
}

// ConvertYAMLToJSON converts YAML to JSON
func (y *YAMLUtils) ConvertYAMLToJSON(yamlData []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to JSON: %w", err)
	}

	return jsonData, nil
}

// ConvertJSONToYAML converts JSON to YAML
func (y *YAMLUtils) ConvertJSONToYAML(jsonData []byte) ([]byte, error) {
	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal to YAML: %w", err)
	}

	return yamlData, nil
}

// Global instances for convenience
var (
	globalJSON = NewJSONUtils()
	globalYAML = NewYAMLUtils()
)

// JSON package-level convenience functions
func MarshalJSON(data interface{}) ([]byte, error) {
	return globalJSON.Marshal(data)
}

func MarshalJSONIndent(data interface{}, prefix, indent string) ([]byte, error) {
	return globalJSON.MarshalIndent(data, prefix, indent)
}

func UnmarshalJSON(data []byte, v interface{}) error {
	return globalJSON.Unmarshal(data, v)
}

func UnmarshalJSONFromReader(reader io.Reader, v interface{}) error {
	return globalJSON.UnmarshalFromReader(reader, v)
}

func MarshalJSONToWriter(writer io.Writer, data interface{}) error {
	return globalJSON.MarshalToWriter(writer, data)
}

func MarshalJSONToFile(filename string, data interface{}) error {
	return globalJSON.MarshalToFile(filename, data)
}

func UnmarshalJSONFromFile(filename string, v interface{}) error {
	return globalJSON.UnmarshalFromFile(filename, v)
}

// YAML package-level convenience functions
func MarshalYAML(data interface{}) ([]byte, error) {
	return globalYAML.Marshal(data)
}

func UnmarshalYAML(data []byte, v interface{}) error {
	return globalYAML.Unmarshal(data, v)
}

func UnmarshalYAMLFromReader(reader io.Reader, v interface{}) error {
	return globalYAML.UnmarshalFromReader(reader, v)
}

func MarshalYAMLToWriter(writer io.Writer, data interface{}) error {
	return globalYAML.MarshalToWriter(writer, data)
}

func MarshalYAMLToFile(filename string, data interface{}) error {
	return globalYAML.MarshalToFile(filename, data)
}

func UnmarshalYAMLFromFile(filename string, v interface{}) error {
	return globalYAML.UnmarshalFromFile(filename, v)
}

func ConvertYAMLToJSON(yamlData []byte) ([]byte, error) {
	return globalYAML.ConvertYAMLToJSON(yamlData)
}

func ConvertJSONToYAML(jsonData []byte) ([]byte, error) {
	return globalYAML.ConvertJSONToYAML(jsonData)
}
