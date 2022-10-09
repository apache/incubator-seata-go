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

import (
	"fmt"
	"time"
)

// Time usable as flag or in YAML config.
type Time time.Time

// String implements flag.Value
func (t Time) String() string {
	if time.Time(t).IsZero() {
		return "0"
	}

	return time.Time(t).Format(time.RFC3339)
}

// Set implements flag.Value
func (t *Time) Set(s string) error {
	if s == "0" {
		*t = Time(time.Time{})
		return nil
	}

	p, err := time.Parse("2006-01-02", s)
	if err == nil {
		*t = Time(p)
		return nil
	}

	p, err = time.Parse("2006-01-02T15:04", s)
	if err == nil {
		*t = Time(p)
		return nil
	}

	p, err = time.Parse("2006-01-02T15:04:05Z07:00", s)
	if err == nil {
		*t = Time(p)
		return nil
	}

	return fmt.Errorf("failed to parse time: %q", s)
}

// UnmarshalYAML implements yaml.Unmarshaler.
func (t *Time) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return t.Set(s)
}

// MarshalYAML implements yaml.Marshaler.
func (t Time) MarshalYAML() (interface{}, error) {
	return t.String(), nil
}
