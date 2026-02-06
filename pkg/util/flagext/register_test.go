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

package flagext_test

import (
	"flag"
	"testing"

	"seata.apache.org/seata-go/v2/pkg/util/flagext"
)

type testRegistererAll struct {
	str        *string
	integer    *int
	int64Val   *int64
	boolean    *bool
	float64Val *float64
}

func (f *testRegistererAll) RegisterFlags(fs *flag.FlagSet) {
	f.str = fs.String("str", "default-string", "string flag")
	f.integer = fs.Int("int", 123, "int flag")
	f.int64Val = fs.Int64("int64", 1234567890, "int64 flag")
	f.boolean = fs.Bool("bool", true, "bool flag")
	f.float64Val = fs.Float64("float", 3.14, "float64 flag")
}

func TestRegisterFlagsAll(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)

	r := &testRegistererAll{}
	flagext.RegisterFlags(r)

	flags := []string{"str", "int", "int64", "bool", "float"}
	for _, name := range flags {
		if flag.CommandLine.Lookup(name) == nil {
			t.Fatalf("expected %s to be registered in flag.CommandLine", name)
		}
	}
}

func TestDefaultValuesAll(t *testing.T) {
	r := &testRegistererAll{}

	flagext.DefaultValues(r)

	if *r.str != "default-string" {
		t.Fatalf("expected default value 'default-string', got %q", *r.str)
	}
	if *r.integer != 123 {
		t.Fatalf("expected default value 123, got %d", *r.integer)
	}
	if *r.int64Val != 1234567890 {
		t.Fatalf("expected default value 1234567890, got %d", *r.int64Val)
	}
	if *r.boolean != true {
		t.Fatalf("expected default value true, got %v", *r.boolean)
	}
	if *r.float64Val != 3.14 {
		t.Fatalf("expected default value 3.14, got %v", *r.float64Val)
	}
}
