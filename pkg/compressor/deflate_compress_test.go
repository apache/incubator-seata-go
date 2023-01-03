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

package compressor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeflateCompress(t *testing.T) {
	ts := []struct {
		text string
	}{
		{
			text: "Don't communicate by sharing memory, share memory by communicating.",
		},
		{
			text: "Concurrency is not parallelism.",
		},
		{
			text: "The bigger the interface, the weaker the abstraction.",
		},
		{
			text: "Documentation is for users.",
		},
	}

	dc := &DeflateCompress{}
	assert.EqualValues(t, CompressorDeflate, dc.GetCompressorType())

	for _, s := range ts {
		var data []byte = []byte(s.text)
		dataCompressed, _ := dc.Compress(data)
		ret, _ := dc.Decompress(dataCompressed)
		assert.EqualValues(t, s.text, string(ret))
	}
}
