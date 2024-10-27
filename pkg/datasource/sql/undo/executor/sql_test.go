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

package executor

import (
	"log"
	"testing"

	"seata.apache.org/seata-go/pkg/datasource/sql/types"

	"github.com/stretchr/testify/assert"
)

// TestDelEscape
func TestDelEscape(t *testing.T) {
	strSlice := []string{`"scheme"."id"`, "`scheme`.`id`", `"scheme".id`, `scheme."id"`, `scheme."id"`, "scheme.`id`"}

	for k, v := range strSlice {
		res := DelEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, "scheme.id", res)
	}
}

// TestAddEscape
func TestAddEscape(t *testing.T) {
	strSlice := []string{`"scheme".id`, "`scheme`.id", `scheme."id"`, "scheme.`id`"}

	for k, v := range strSlice {
		res := AddEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, v, res)
	}

	strSlice1 := []string{"ALTER", "ANALYZE"}
	for k, v := range strSlice1 {
		res := AddEscape(v, types.DBTypeMySQL)
		log.Printf("val_%d: %s, res_%d: %s\n", k, v, k, res)
		assert.Equal(t, "`"+v+"`", res)
	}
}
