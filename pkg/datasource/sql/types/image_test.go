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

package types

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestColumnImage(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, time.Now().String())
	values := []interface{}{
		[]byte("test"),
		true,
		false,
		byte(10),
		rune(100),
		int(10),
		uint(10),
		int8(10),
		uint8(10),
		int64(10),
		uint16(10),
		int32(10),
		uint32(10),
		int64(10),
		uint64(10),
		float32(10),
		float64(10),
		now,
		string("test"),
	}
	for _, val := range values {
		t.Run(fmt.Sprintf("%T", val), func(t *testing.T) {
			c := ColumnImage{
				ColumnName: "test",
				Value:      val,
			}
			cBytes, err := json.Marshal(&c)
			if err != nil {
				t.Errorf("Marshal error:%v", err)
			}
			cT := ColumnImage{}
			err = json.Unmarshal(cBytes, &cT)
			if err != nil {
				t.Errorf("Unmarshal error:%v", err)
			}
			assert.Equal(t, c.Value, cT.Value)
		})
	}
}
